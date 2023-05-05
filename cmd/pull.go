package cmd

import (
	"bytes"
	"fmt"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var providerPath = ""

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pulls description fields from markdown documentation into schema",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("pull called")

		//dat, err := os.ReadFile("../terraform-provider-aws/internal/service/amp/workspace.go")
		//fset := token.NewFileSet()
		//file, err := parser.ParseFile(fset, "workspace.go", dat, parser.AllErrors)
		//if err != nil {
		//	log.Fatal(err)
		//}
		log.Println("loaded")

		providerPath, _ = filepath.Abs(providerPath)

		// Find and iterate through resource and datasource files

		iterateThroughFiles(fmt.Sprintf("%s/internal/service/", providerPath), pull)

	},
}

type pullFile func(path string, contents string, opts pullOptions)

func pull(path string, contents string, opts pullOptions) {

	docFilePath := findDocumentationFile(contents, opts)

	if _, err := os.Stat(docFilePath); err != nil {
		log.Printf("No documentation file found for %s", docFilePath)
		return
	}

	file, err := decorator.Parse(contents)
	if err != nil {
		log.Fatalf(err.Error())
	}

	applyFunc := func(c *dstutil.Cursor) bool {
		node := c.Node()
		switch n := node.(type) {
		case *dst.KeyValueExpr:
			key, ok := n.Key.(*dst.BasicLit)
			if ok {
				attrName := key.Value
				attrName = attrName[1 : len(attrName)-1]

				doc, err := os.ReadFile(docFilePath)
				if err != nil {
					log.Fatal(err)
				}
				re := fmt.Sprintf("\\* `%v` - (.*)", attrName)
				r, err := regexp.Compile(re)

				if err != nil {
					fmt.Printf("Failed to create regex for file %s and attribute: %s", path, attrName)
					return false
				}

				matches := r.FindAllStringSubmatch(string(doc), -1)

				if len(matches) > 0 {
					val, valOk := n.Value.(*dst.CompositeLit)
					if valOk {
						kve := dst.KeyValueExpr{}
						kve.Key = &dst.Ident{Name: "Description"}
						kve.Value = &dst.Ident{Name: fmt.Sprintf("\"%s\"", matches[0][1])}
						kve.Decorations().Before = dst.NewLine
						kve.Decorations().After = dst.NewLine
						val.Elts = append([]dst.Expr{&kve}, val.Elts...)
					}
				}
			}
		}
		return true
	}
	_ = dstutil.Apply(file, applyFunc, nil)

	newContents := FormatNode(*file)

	//fmt.Println(newContents)

	distPath := strings.Replace(path, "terraform-provider-aws/internal", "terraform-provider-aws/dist/internal", -1)

	err = os.MkdirAll(filepath.Dir(distPath), 0777)
	if err != nil {
		return
	}
	f, err := os.Create(distPath)
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(newContents)
	if err != nil {
		log.Printf(newContents)
		log.Fatal(err)
	}

	//panic("")

	//docFilePath := findDocumentationFile(contents, opts)
	//altFilePath := strings.Replace(docFilePath, "html.markdown", "markdown", -1)
	//
	//if _, err := os.Stat(docFilePath); err != nil {
	//	if _, err := os.Stat(altFilePath); err != nil {
	//		fmt.Printf("%s does not exist\n", docFilePath)
	//	}
	//}
}
func FormatNode(file dst.File) string {
	var buf bytes.Buffer
	decorator.Fprint(&buf, &file)
	return buf.String()
}

type pullOptions struct {
	isFramework  bool
	isDataSource bool
}

func iterateThroughFiles(path string, fn pullFile) {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf(err.Error())
		}
		if strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
			contentsBytes, _ := os.ReadFile(path)
			contents := string(contentsBytes)
			var opt pullOptions
			var fileTypeFound = true
			if strings.Contains(contents, "// @SDKDataSource") {
				opt.isFramework = false
				opt.isDataSource = true
			} else if strings.Contains(contents, "// @FrameworkDataSource") {
				opt.isFramework = true
				opt.isDataSource = true
			} else if strings.Contains(contents, "// @SDKResource") {
				opt.isFramework = false
				opt.isDataSource = false
			} else if strings.Contains(contents, "// @FrameworkResource") {
				opt.isFramework = true
				opt.isDataSource = false
			} else {
				fileTypeFound = false
			}
			if fileTypeFound {
				fn(path, contents, opt)
			}
		}
		return nil
	})
	if err != nil {
		return
	}
}

/*func findPackageName(contents string) string {
	r := regexp.MustCompile("package (\\w*)")
	matches := r.FindAllStringSubmatch(contents, -1)
	return matches[0][1]
}*/

func findDocumentationFile(contents string, opts pullOptions) string {
	docTypePrefix := ""
	var r *regexp.Regexp

	if opts.isFramework {
		r = regexp.MustCompile(`.TypeName = "(\w*)"`)
	} else {
		r = regexp.MustCompile(`(?m)\/\/ @SDK(Resource|DataSource)\("(\w*)`)
	}

	if opts.isDataSource {
		docTypePrefix = "d"
	} else {
		docTypePrefix = "r"
	}

	matches := r.FindAllStringSubmatch(contents, -1)

	var docFileName string
	if opts.isFramework {
		docFileName = matches[0][1]
	} else {
		docFileName = matches[0][2]
	}

	docFileName = strings.TrimPrefix(docFileName, "aws_")

	docFileName = processExceptions(docFileName)

	return filepath.Join(providerPath, fmt.Sprintf("website/docs/%s/%s.html.markdown", docTypePrefix, docFileName))
}

func processExceptions(filename string) string {
	lbExceptions := []string{"alb_listener", "alb_listener_certificate", "alb_listener", "alb_listener_rule", "alb", "alb_target_group", "alb_target_group_attachment"}
	if slices.Contains(lbExceptions, filename) {
		filename = strings.TrimPrefix(filename, "alb")
		filename = "lb" + filename
	}
	return filename
}

func init() {
	rootCmd.AddCommand(pullCmd)

	pullCmd.Flags().StringVarP(&providerPath, "provider-path", "p", "../terraform-provider-aws", "Path to AWS Provider")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pullCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pullCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

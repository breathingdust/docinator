/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
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

		////file := "workspace.go"
		//exDocFilePath := "../terraform-provider-aws/website/docs/r/prometheus_workspace.html.markdown"
		//
		//astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		//	n := c.Node()
		//	switch x := n.(type) {
		//	case *ast.KeyValueExpr:
		//		key, ok := x.Key.(*ast.BasicLit)
		//		if ok {
		//			attrName := key.Value
		//			attrName = attrName[1 : len(attrName)-1]
		//
		//			fmt.Println(attrName)
		//
		//			doc, err := os.ReadFile(exDocFilePath)
		//			if err != nil {
		//				log.Fatal(err)
		//			}
		//			re := fmt.Sprintf("\\* `%v` - (.*)", attrName)
		//			fmt.Println(re)
		//			r, _ := regexp.Compile(re)
		//
		//			description := r.FindString(string(doc))
		//
		//			val, valOk := x.Value.(*ast.CompositeLit)
		//			if valOk {
		//				kve := ast.KeyValueExpr{}
		//				kve.Key = &ast.Ident{Name: "Description"}
		//				kve.Value = &ast.Ident{Name: fmt.Sprintf("\"%s\"", description)}
		//				val.Elts = append([]ast.Expr{&kve}, val.Elts...)
		//
		//				val.Elts[0] = c.InsertBefore(kve, val.Elts[0])
		//			}
		//
		//			fmt.Println(description)
		//
		//		}
		//	}
		//
		//	// look for a basiclit where there is aprent kve
		//
		//	return true
		//})
		//var buf bytes.Buffer
		//err = printer.Fprint(&buf, fset, file)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//err = os.WriteFile("test.go", buf.Bytes(), 0644)
		//if err != nil {
		//	log.Fatal(err)
		//}
	},
}

type pullFile func(filename string, contents string, opts pullOptions)

func pull(filename string, contents string, opts pullOptions) {
	//fset := token.NewFileSet()
	//file, err := parser.ParseFile(fset, filename, contents, parser.AllErrors)
	//if err != nil {
	//	log.Fatal(err)
	//}

	docFilePath := findDocumentationFile(contents, opts)
	altFilePath := strings.Replace(docFilePath, "html.markdown", "markdown", -1)

	if _, err := os.Stat(docFilePath); err != nil {
		if _, err := os.Stat(altFilePath); err != nil {
			fmt.Printf("%s does not exist\n", docFilePath)
		}
	}
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
				fn(info.Name(), contents, opt)
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

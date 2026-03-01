package examples_test

import (
	"fmt"
	"os"

	"github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_html_parse_file() {
	f, err := os.CreateTemp("", "helium-html-*.html")
	if err != nil {
		fmt.Printf("create temp file failed: %s\n", err)
		return
	}
	defer os.Remove(f.Name()) //nolint:errcheck

	if _, err := f.WriteString(`<!doctype html><html><body><h1>FromFile</h1></body></html>`); err != nil {
		fmt.Printf("write temp file failed: %s\n", err)
		return
	}
	if err := f.Close(); err != nil {
		fmt.Printf("close temp file failed: %s\n", err)
		return
	}

	doc, err := html.ParseFile(f.Name())
	if err != nil {
		fmt.Printf("parse file failed: %s\n", err)
		return
	}

	nodes, err := xpath.Find(doc, `//h1`)
	if err != nil {
		fmt.Printf("xpath failed: %s\n", err)
		return
	}

	fmt.Println(string(nodes[0].Content()))
	// Output:
	// FromFile
}

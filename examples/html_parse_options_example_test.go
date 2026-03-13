package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/html"
)

func Example_html_parse_options() {
	doc, err := html.Parse(context.Background(), []byte("<div>\n  <span>hi</span>\n</div>"),
		html.WithNoImplied(),
		html.WithNoBlanks(),
	)
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	root := doc.DocumentElement()
	fmt.Println(root.Name())

	children := 0
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		children++
	}
	fmt.Println(children)
	// Output:
	// div
	// 1
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/html"
)

func Example_html_parse_options() {
	// Parse options let you tune how forgiving the HTML parser should be.
	// WithNoImplied keeps the parser from synthesizing html/head/body elements,
	// and WithNoBlanks drops whitespace-only text nodes from the DOM.
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

	// Because WithNoBlanks removed the indentation-only text node, the <div>
	// has just one child element here.
	children := 0
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		children++
	}
	fmt.Println(children)
	// Output:
	// div
	// 1
}

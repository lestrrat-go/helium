package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_html_parse() {
	// html.Parse builds a helium DOM from HTML input and applies HTML-specific
	// parsing rules (implied elements, case-insensitive tag handling, etc.).
	doc, err := html.Parse([]byte(`<h1>Title</h1><div>Hello</div>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// The parsed document uses the HTML document node type.
	fmt.Println(doc.Type() == helium.HTMLDocumentNode)

	// Parsed HTML can be queried with regular XPath helpers.
	nodes, err := xpath.Find(doc, `//div`)
	if err != nil {
		fmt.Printf("xpath failed: %s\n", err)
		return
	}
	fmt.Println(len(nodes))
	fmt.Println(string(nodes[0].Content()))
	// Output:
	// true
	// 1
	// Hello
}

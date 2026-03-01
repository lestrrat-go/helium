package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_xpath_find() {
	doc, err := helium.Parse([]byte(`<catalog><book id="1">Go</book><book id="2">XML</book><magazine/></catalog>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// xpath.Find is a convenience function that evaluates an XPath expression
	// and returns the resulting node set directly. It is a shorthand for
	// calling Evaluate and accessing the NodeSet field of the result.
	// The expression "//book" selects all <book> elements anywhere in the
	// document tree.
	nodes, err := xpath.Find(doc, "//book")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}

	fmt.Printf("found %d nodes\n", len(nodes))
	for _, n := range nodes {
		// Name returns the element's local name, and Content returns
		// the concatenated text content of the element and its descendants.
		fmt.Printf("  %s: %s\n", n.Name(), string(n.Content()))
	}
	// Output:
	// found 2 nodes
	//   book: Go
	//   book: XML
}

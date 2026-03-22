package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_find() {
	doc, err := helium.Parse(context.Background(), []byte(`<catalog><book id="1">Go</book><book id="2">XML</book><magazine/></catalog>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// xpath3.Find is the convenience API for the common "give me the matching
	// nodes" case. It compiles the XPath expression, evaluates it, and returns
	// the node slice directly.
	nodes, err := xpath3.Find(context.Background(), doc, "//book")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}

	fmt.Printf("found %d nodes\n", len(nodes))
	for _, n := range nodes {
		fmt.Printf("  %s: %s\n", n.Name(), string(n.Content()))
	}
	// Output:
	// found 2 nodes
	//   book: Go
	//   book: XML
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_with_namespaces() {
	const src = `<root xmlns:ns="http://example.com/ns"><ns:item>one</ns:item><ns:item>two</ns:item></root>`

	doc, err := helium.Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	ctx := xpath3.WithNamespaces(context.Background(), map[string]string{
		"x": "http://example.com/ns",
	})

	// XPath expressions do not automatically know the namespace prefixes used in
	// your Go code. Bind the prefixes you want to use in the expression through
	// the context, then evaluate as normal.
	r, err := xpath3.Evaluate(ctx, doc, "//x:item")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	nodes, err := r.Nodes()
	if err != nil {
		fmt.Printf("unexpected non-node result: %s\n", err)
		return
	}
	fmt.Printf("found %d nodes\n", len(nodes))
	for _, n := range nodes {
		fmt.Printf("  %s\n", string(n.Content()))
	}
	// Output:
	// found 2 nodes
	//   one
	//   two
}

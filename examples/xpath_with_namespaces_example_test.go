package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

func Example_xpath_with_namespaces() {
	// The document uses prefix "ns" for the namespace URI.
	const src = `<root xmlns:ns="http://example.com/ns"><ns:item>one</ns:item><ns:item>two</ns:item></root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// XPath expressions need namespace prefixes to match namespaced elements,
	// but the prefixes used in XPath don't have to match those in the document.
	// Here we bind prefix "x" to the same namespace URI that the document
	// uses with prefix "ns". This lets us write "//x:item" in the XPath
	// expression to match <ns:item> elements.
	//
	// This decoupling is important because XPath matches by namespace URI,
	// not by prefix — documents may use any prefix for a given namespace.
	ev := xpath1.NewEvaluator().Namespaces(map[string]string{
		"x": "http://example.com/ns",
	})

	// Evaluate with explicit namespace bindings via Evaluator.
	expr, err := xpath1.Compile("//x:item")
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := ev.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("found %d nodes\n", len(r.NodeSet))
	for _, n := range r.NodeSet {
		fmt.Printf("  %s\n", string(n.Content()))
	}
	// Output:
	// found 2 nodes
	//   one
	//   two
}

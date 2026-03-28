package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_with_namespaces() {
	const src = `<root xmlns:ns="http://example.com/ns"><ns:item>one</ns:item><ns:item>two</ns:item></root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// XPath expressions do not automatically know the namespace prefixes used in
	// your Go code. Bind the prefixes you want to use in the expression through
	// the Evaluator, then evaluate as normal.
	compiled, err := xpath3.NewCompiler().Compile("//x:item")
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	// Create an evaluator and bind the prefix "x" to the namespace URI used
	// in the document. The prefix in the XPath expression ("x:item") does not
	// have to match the prefix in the source XML ("ns:item") — only the URI
	// must match.
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(map[string]string{
			"x": "http://example.com/ns",
		})

	r, err := eval.Evaluate(context.Background(), compiled, doc)
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

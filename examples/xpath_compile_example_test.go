package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

func Example_xpath_compile() {
	// xpath1.Compile pre-compiles an XPath expression so it can be
	// evaluated multiple times without re-parsing. This is useful
	// when the same expression needs to be applied to many nodes.
	expr, err := xpath1.Compile("count(child::*)")
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	// String() returns the original expression text.
	fmt.Printf("expression: %s\n", expr.String())

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><a><x/><y/></a><b><z/></b></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Evaluate the compiled expression against each top-level child.
	// "count(child::*)" counts the direct child elements of the context node.
	// <a> has 2 children (x, y) and <b> has 1 child (z).
	ev := xpath1.NewEvaluator()
	for child := doc.FirstChild().FirstChild(); child != nil; child = child.NextSibling() {
		r, err := ev.Evaluate(context.Background(), expr, child)
		if err != nil {
			fmt.Printf("eval error: %s\n", err)
			return
		}
		fmt.Printf("%s has %.0f children\n", child.Name(), r.Number)
	}
	// Output:
	// expression: count(child::*)
	// a has 2 children
	// b has 1 children
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_xpath_evaluate() {
	doc, err := helium.Parse(context.Background(), []byte(`<prices><item>10</item><item>20</item><item>30</item></prices>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// xpath.Evaluate returns a typed Result that can hold different XPath
	// result types: node sets, strings, numbers, or booleans.
	// The result type depends on the XPath expression used.

	// string() XPath function converts a node's text content to a string.
	// The result is available in the String field.
	r, err := xpath.Evaluate(context.Background(), doc,"string(//item[1])")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("string: %s\n", r.String)

	// sum() XPath function adds up the numeric values of all nodes
	// in the node set. The result is available in the Number field.
	r, err = xpath.Evaluate(context.Background(), doc,"sum(//item)")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("sum: %.0f\n", r.Number)

	// Comparison expressions return a boolean result, available
	// in the Bool field.
	r, err = xpath.Evaluate(context.Background(), doc,"count(//item) > 2")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("more than 2: %t\n", r.Bool)
	// Output:
	// string: 10
	// sum: 60
	// more than 2: true
}

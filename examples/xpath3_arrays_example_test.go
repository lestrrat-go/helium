package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_arrays() {
	// XPath 3.1 arrays are ordered sequences of values. Construct one with
	// square brackets and query it with array:size, the lookup operator (?),
	// and functions like array:flatten.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(map[string]string{
			"array": "http://www.w3.org/2005/xpath-functions/array",
		})

	// Get the size of an array.
	expr, err := compiler.Compile(`array:size(["a", "b", "c"])`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	n, _ := r.IsNumber()
	fmt.Printf("size: %.0f\n", n)

	// Look up a value by 1-based index using the ? operator.
	expr, err = compiler.Compile(`["Go", "XML", "XPath"]?2`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	s, _ := r.IsString()
	fmt.Printf("item 2: %s\n", s)

	// Flatten a nested array into a flat sequence.
	expr, err = compiler.Compile(`string-join(array:flatten([["a", "b"], ["c"]]), ",")`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	flat, _ := r.IsString()
	fmt.Printf("flattened: %s\n", flat)

	// Output:
	// size: 3
	// item 2: XML
	// flattened: a,b,c
}

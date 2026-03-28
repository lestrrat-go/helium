package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_maps() {
	// XPath 3.1 maps are key-value structures. Construct one with map { ... }
	// and query it with map:size, map:keys, and the lookup operator (?).
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Namespaces(map[string]string{
			"map": "http://www.w3.org/2005/xpath-functions/map",
		})

	// Build a map and get its size.
	expr, err := compiler.Compile(`map:size(map { "x": 1, "y": 2, "z": 3 })`)
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

	// Look up a value by key using the ? operator.
	expr, err = compiler.Compile(`map { "lang": "Go", "year": 2009 }?lang`)
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
	fmt.Printf("lang: %s\n", s)

	// Use map:put to add a key and map:contains to test membership.
	expr, err = compiler.Compile(`map:contains(map:put(map { "a": 1 }, "b", 2), "b")`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	b, _ := r.IsBoolean()
	fmt.Printf("contains b: %t\n", b)

	// Output:
	// size: 3
	// lang: Go
	// contains b: true
}

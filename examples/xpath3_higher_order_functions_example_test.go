package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_higher_order_functions() {
	// XPath 3.1 supports inline functions and higher-order functions like
	// for-each and filter that accept functions as arguments.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	// Use for-each to apply a function to each item in a sequence.
	// upper-case#1 is a function reference (name + arity).
	expr, err := compiler.Compile(`string-join(for-each(("go", "xml"), upper-case#1), " ")`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	s, _ := r.IsString()
	fmt.Printf("for-each: %s\n", s)

	// Use filter to keep only items matching a predicate.
	expr, err = compiler.Compile(`string-join(filter(("apple", "banana", "avocado"), function($s) { starts-with($s, "a") }), ",")`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	s, _ = r.IsString()
	fmt.Printf("filter: %s\n", s)

	// Use fold-left to accumulate a sum.
	expr, err = compiler.Compile(`fold-left((1, 2, 3, 4), 0, function($acc, $x) { $acc + $x })`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	n, _ := r.IsNumber()
	fmt.Printf("fold-left sum: %.0f\n", n)

	// Output:
	// for-each: GO XML
	// filter: apple,avocado
	// fold-left sum: 10
}

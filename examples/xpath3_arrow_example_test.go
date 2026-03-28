package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_arrow_operator() {
	// The arrow operator (=>) pipes a value into a function as the first
	// argument, enabling a readable left-to-right pipeline style.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	// Chain string transformations: normalize → uppercase → take first 5 chars.
	expr, err := compiler.Compile(`"  hello world  " => normalize-space() => upper-case() => substring(1, 5)`)
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
	fmt.Printf("piped: %s\n", s)

	// Use => with string-join to format a sequence.
	expr, err = compiler.Compile(`(1, 2, 3) => string-join("-")`)
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
	fmt.Printf("joined: %s\n", s)

	// Output:
	// piped: HELLO
	// joined: 1-2-3
}

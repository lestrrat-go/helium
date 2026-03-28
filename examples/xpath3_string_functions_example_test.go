package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_string_functions() {
	// XPath 3.1 includes powerful string functions for regex matching,
	// replacement, and tokenization.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	// matches() — test whether a string matches a regex.
	expr, err := compiler.Compile(`matches("2024-01-15", "^\d{4}-\d{2}-\d{2}$")`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	b, _ := r.IsBoolean()
	fmt.Printf("is date: %t\n", b)

	// replace() — regex replacement.
	expr, err = compiler.Compile(`replace("Hello World 123", "\d+", "###")`)
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
	fmt.Printf("replaced: %s\n", s)

	// tokenize() — split a string by regex.
	expr, err = compiler.Compile(`string-join(tokenize("go,xml, xpath", "[,\s]+"), " | ")`)
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
	fmt.Printf("tokens: %s\n", s)

	// Output:
	// is date: true
	// replaced: Hello World ###
	// tokens: go | xml | xpath
}

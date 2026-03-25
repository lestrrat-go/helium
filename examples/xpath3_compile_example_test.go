package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_compile() {
	// xpath3.Compile pre-compiles an XPath 3.1 expression so it can be
	// evaluated multiple times without re-parsing.
	expr, err := xpath3.NewCompiler().Compile("count(child::*)")
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	fmt.Printf("expression: %s\n", expr.String())

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><a><x/><y/></a><b><z/></b></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	for child := doc.FirstChild().FirstChild(); child != nil; child = child.NextSibling() {
		r, err := eval.Evaluate(context.Background(), expr, child)
		if err != nil {
			fmt.Printf("eval error: %s\n", err)
			return
		}
		n, ok := r.IsNumber()
		if !ok {
			fmt.Println("unexpected non-numeric result")
			return
		}
		fmt.Printf("%s has %.0f children\n", child.Name(), n)
	}
	// Output:
	// expression: count(child::*)
	// a has 2 children
	// b has 1 children
}

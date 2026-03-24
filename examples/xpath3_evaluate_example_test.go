package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_evaluate() {
	doc, err := helium.Parse(context.Background(), []byte(`<prices><item>10</item><item>20</item><item>30</item></prices>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Compile expressions individually and evaluate with a reusable evaluator.
	// The Result wrapper exposes typed accessors that match the expression you ran.
	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	expr, err := compiler.Compile("string(//item[1])")
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	s, ok := r.IsString()
	if !ok {
		fmt.Println("unexpected non-string result")
		return
	}
	fmt.Printf("string: %s\n", s)

	// The same document can be queried again for a numeric result.
	expr, err = compiler.Compile("sum(//item)")
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	n, ok := r.IsNumber()
	if !ok {
		fmt.Println("unexpected non-numeric result")
		return
	}
	fmt.Printf("sum: %.0f\n", n)

	// And again for a boolean result.
	expr, err = compiler.Compile("count(//item) > 2")
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	b, ok := r.IsBoolean()
	if !ok {
		fmt.Println("unexpected non-boolean result")
		return
	}
	fmt.Printf("more than 2: %t\n", b)
	// Output:
	// string: 10
	// sum: 60
	// more than 2: true
}

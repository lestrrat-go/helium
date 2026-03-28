package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

func Example_xpath_context_options() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<catalog><book price="30"/><book price="45"/></catalog>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Evaluator options let you inject external values into XPath evaluation.
	// Each builder method returns a new evaluator, so you can chain or
	// assign intermediate steps.
	ev := xpath1.NewEvaluator()

	// Bind a variable that XPath expressions can reference as $minPrice.
	ev = ev.Variables(map[string]any{
		"minPrice": float64(40),
	})

	// Register a custom XPath function named "discount" that multiplies
	// its numeric argument by 0.9.
	ev = ev.Function("discount", xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{
			Type:   xpath1.NumberResult,
			Number: args[0].Number * 0.9,
		}, nil
	}))

	// The first expression reads the caller-supplied variable.
	expr1, err := xpath1.Compile(`count(/catalog/book[number(@price) >= $minPrice])`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := ev.Evaluate(context.Background(), expr1, doc)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("eligible: %.0f\n", r.Number)

	// The second expression calls the custom XPath function registered above.
	expr2, err := xpath1.Compile(`discount(number(/catalog/book[1]/@price))`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = ev.Evaluate(context.Background(), expr2, doc)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("discounted: %.0f\n", r.Number)
	// Output:
	// eligible: 1
	// discounted: 27
}

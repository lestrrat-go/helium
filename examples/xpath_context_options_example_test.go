package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

func Example_xpath_context_options() {
	doc, err := helium.Parse(context.Background(), []byte(`<catalog><book price="30"/><book price="45"/></catalog>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	ctx := xpath1.NewContext(context.Background(),
		xpath1.WithVariables(map[string]any{
			"minPrice": float64(40),
		}),
	)

	xctx := xpath1.GetContext(ctx)
	if err := xctx.RegisterFunction("discount", xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{
			Type:   xpath1.NumberResult,
			Number: args[0].Number * 0.9,
		}, nil
	})); err != nil {
		fmt.Printf("failed to register function: %s\n", err)
		return
	}

	r, err := xpath1.Evaluate(ctx, doc, `count(/catalog/book[number(@price) >= $minPrice])`)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("eligible: %.0f\n", r.Number)

	r, err = xpath1.Evaluate(ctx, doc, `discount(number(/catalog/book[1]/@price))`)
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	fmt.Printf("discounted: %.0f\n", r.Number)
	// Output:
	// eligible: 1
	// discounted: 27
}

package xpath3_test

import (
	"context"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// evaluate is a test helper: compile + evaluate in one call.
func evaluate(ctx context.Context, node helium.Node, expr string) (*xpath3.Result, error) {
	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		return nil, err
	}
	return xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, node)
}

// find is a test helper: compile + evaluate, returning a node-set.
func find(ctx context.Context, node helium.Node, expr string) ([]helium.Node, error) {
	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		return nil, err
	}
	r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, node)
	if err != nil {
		return nil, err
	}
	return r.Nodes()
}

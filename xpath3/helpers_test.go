package xpath3_test

import (
	"context"
	"math/big"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// Test-fixture-only literals (no lexicon equivalent).
const (
	// Frequently reused literal values in test tables.
	testHello    = "hello"
	testFoo      = "foo"
	testMutated  = "mutated"
	testValue    = "test"
	testExpr1Ne2 = "1 != 2"
	// paramsVar is the variable name bound to serialization-parameters elements
	// in fn:serialize element-options tests.
	paramsVar = "params"
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

// Shared test-only string constants that recur across the package's test files.
// Centralizing them keeps goconst happy and avoids drift between expectations.
const (
	wantTrue  = "true"
	wantFalse = "false"
	wantNaN   = "NaN"
	wantINF   = "INF"
	want1Dot5 = "1.5"
	expr1To10 = "1 to 10"
)

func atomicSeq(av xpath3.AtomicValue) xpath3.Sequence {
	return xpath3.ItemSlice{av}
}

func intAtomic(n int64) xpath3.AtomicValue {
	return xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(n)}
}

func strAtomic(s string) xpath3.AtomicValue {
	return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}
}

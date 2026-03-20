package xpath3

import (
	"context"

	"github.com/lestrrat-go/helium"
)

// NewLexerForTesting exposes the internal lexer for tests.
func NewLexerForTesting(input string) (*lexer, error) {
	return newLexer(input)
}

// EvalForTesting evaluates a parsed expression against a node for testing.
func EvalForTesting(ctx context.Context, node helium.Node, expr Expr) (Sequence, error) {
	compiled, err := CompileExpr(expr)
	if err != nil {
		return nil, err
	}
	ec := newEvalContext(ctx, node)
	return compiled.evaluate(ec)
}

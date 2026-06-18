package xpath3

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUnarySchemaDerivedNumeric verifies that unary plus and minus promote a
// schema-derived numeric operand (custom TypeName with a built-in numeric
// BaseType) to its built-in base before the numeric type check and negation.
// Before the fix the operand's custom TypeName failed isSubtypeOf/TypeName checks
// and both +$x and -$x raised XPTY0004.
func TestUnarySchemaDerivedNumeric(t *testing.T) {
	// stubEval returns the supplied operand regardless of the Expr/context, so we
	// can drive evalUnaryExpr directly with a schema-derived value.
	stubEval := func(a AtomicValue) exprEvaluator {
		return func(context.Context, *evalContext, Expr) (Sequence, error) {
			return SingleAtomic(a), nil
		}
	}

	unary := func(t *testing.T, a AtomicValue, negate bool) AtomicValue {
		t.Helper()
		got, err := evalUnaryExpr(stubEval(a), t.Context(), nil, UnaryExpr{Operand: LiteralExpr{}, Negate: negate})
		require.NoError(t, err)
		require.Equal(t, 1, seqLen(got))
		av, ok := got.(ItemSlice)[0].(AtomicValue)
		require.True(t, ok)
		return av
	}

	t.Run("derived integer", func(t *testing.T) {
		x := AtomicValue{TypeName: "my:int", BaseType: TypeInteger, Value: int64(2)}
		require.Equal(t, 2.0, unary(t, x, false).ToFloat64())
		neg := unary(t, x, true)
		require.True(t, isIntegerDerived(neg.TypeName))
		require.Equal(t, int64(-2), neg.BigInt().Int64())
	})

	t.Run("derived decimal", func(t *testing.T) {
		x := AtomicValue{TypeName: "my:dec", BaseType: TypeDecimal, Value: big.NewRat(3, 2)}
		require.Equal(t, 1.5, unary(t, x, false).ToFloat64())
		neg := unary(t, x, true)
		require.Equal(t, TypeDecimal, neg.TypeName)
		require.Equal(t, 0, neg.BigRat().Cmp(big.NewRat(-3, 2)))
	})

	t.Run("derived float", func(t *testing.T) {
		x := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: NewFloat(2)}
		pos := unary(t, x, false)
		require.Equal(t, 2.0, pos.ToFloat64())
		neg := unary(t, x, true)
		require.Equal(t, TypeFloat, neg.TypeName)
		require.Equal(t, -2.0, neg.ToFloat64())
	})
}

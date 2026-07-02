package xpath3

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// evalCompat compiles and evaluates expr with XPath 1.0 compatibility mode on.
func evalCompat(t *testing.T, expr string) *Result {
	t.Helper()
	e, err := NewCompiler().Compile(expr)
	require.NoError(t, err)
	res, err := NewEvaluator(DefaultEvaluatorOptions).XPath10Compat().Evaluate(t.Context(), e, nil)
	require.NoError(t, err)
	return res
}

// evalPlain compiles and evaluates expr with ordinary (non-compat) semantics.
func evalPlain(t *testing.T, expr string) (*Result, error) {
	t.Helper()
	e, err := NewCompiler().Compile(expr)
	require.NoError(t, err)
	return NewEvaluator(DefaultEvaluatorOptions).Evaluate(t.Context(), e, nil)
}

func TestXPath10CompatArithmetic(t *testing.T) {
	num := func(t *testing.T, expr string) float64 {
		t.Helper()
		n, ok := evalCompat(t, expr).IsNumber()
		require.True(t, ok, "expected numeric result for %q", expr)
		return n
	}

	t.Run("integer div by zero yields +INF", func(t *testing.T) {
		require.True(t, math.IsInf(num(t, "1 div 0"), 1))
	})
	t.Run("negative div by zero yields -INF", func(t *testing.T) {
		require.True(t, math.IsInf(num(t, "-1 div 0"), -1))
	})
	t.Run("string operand cast to double", func(t *testing.T) {
		require.Equal(t, 7.0, num(t, "'3' + 4"))
	})
	t.Run("non-numeric operand yields NaN", func(t *testing.T) {
		require.True(t, math.IsNaN(num(t, "'x' + 1")))
	})
	t.Run("empty operand yields NaN", func(t *testing.T) {
		require.True(t, math.IsNaN(num(t, "() + 1")))
	})
	t.Run("unary minus of string", func(t *testing.T) {
		require.Equal(t, -5.0, num(t, "-'5'"))
	})
	t.Run("integer arithmetic result is double", func(t *testing.T) {
		// 12 div 5 = 2.4 (double), not integer/decimal truncation.
		require.Equal(t, 2.4, num(t, "12 div 5"))
	})
}

func TestXPath10CompatGeneralComparison(t *testing.T) {
	boolOf := func(t *testing.T, expr string) bool {
		t.Helper()
		b, ok := evalCompat(t, expr).IsBoolean()
		require.True(t, ok, "expected boolean result for %q", expr)
		return b
	}

	t.Run("numeric string greater than integer", func(t *testing.T) {
		require.True(t, boolOf(t, "'35' > 17"))
	})
	t.Run("integer less than numeric string", func(t *testing.T) {
		require.True(t, boolOf(t, "6 < '44'"))
	})
	t.Run("empty string equals boolean via EBV", func(t *testing.T) {
		// EBV('') = false; false = true() -> false
		require.False(t, boolOf(t, "'' = true()"))
	})
	t.Run("non-empty string equals boolean via EBV", func(t *testing.T) {
		// EBV('x') = true; true = true() -> true
		require.True(t, boolOf(t, "'x' = true()"))
	})
	t.Run("string equality still works", func(t *testing.T) {
		require.True(t, boolOf(t, "'abc' = 'abc'"))
	})
	t.Run("relational op with boolean operand compares as numbers, not EBV", func(t *testing.T) {
		// XPath 1.0: relational operators always convert both operands to number,
		// so true() < 5 is number(true())=1 < 5 = true (NOT EBV(true()) < EBV(5)).
		require.True(t, boolOf(t, "true() < 5"))
		require.False(t, boolOf(t, "true() > 5"))
	})
	t.Run("non-numeric string vs number is false", func(t *testing.T) {
		// number('abc') = NaN, NaN = 5 -> false
		require.False(t, boolOf(t, "'abc' = 5"))
	})
}

func TestXPath10CompatFunctionConversion(t *testing.T) {
	t.Run("string of multi-item sequence takes first", func(t *testing.T) {
		s, ok := evalCompat(t, "string((1, 2, 3))").IsString()
		require.True(t, ok)
		require.Equal(t, "1", s)
	})
	t.Run("number of multi-item sequence takes first", func(t *testing.T) {
		n, ok := evalCompat(t, "number(('4', 'x'))").IsNumber()
		require.True(t, ok)
		require.Equal(t, 4.0, n)
	})
	t.Run("string-typed arg to double param is coerced via fn:number", func(t *testing.T) {
		// substring's start arg is xs:double; a plain xs:string '2' would be
		// XPTY0004 without compat, but here it is fn:number-coerced.
		s, ok := evalCompat(t, "substring('abcde', '2')").IsString()
		require.True(t, ok)
		require.Equal(t, "bcde", s)
	})
}

// TestXPath10CompatOptIn confirms the mode is off by default: the same
// expressions that succeed under compat raise type errors in ordinary mode.
func TestXPath10CompatOptIn(t *testing.T) {
	t.Run("string of multi-item sequence errors without compat", func(t *testing.T) {
		_, err := evalPlain(t, "string((1, 2, 3))")
		require.Error(t, err)
	})
	t.Run("string arg to double param errors without compat", func(t *testing.T) {
		_, err := evalPlain(t, "substring('abcde', '2')")
		require.Error(t, err)
	})
}

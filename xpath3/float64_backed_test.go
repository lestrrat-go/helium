package xpath3

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// float64Double builds an xs:double atomic backed by a plain float64 value 2.0
// (not a *FloatValue), as a schema-derived or directly constructed value can be.
func float64Double() AtomicValue {
	return AtomicValue{TypeName: TypeDouble, Value: float64(2)}
}

// TestFloat64BackedNoPanic verifies that no xpath3 code path panics on an
// xs:double atomic whose backing value is a plain float64 rather than a
// *FloatValue. Several call sites used to force-assert *FloatValue.
func TestFloat64BackedNoPanic(t *testing.T) {
	t.Run("DoubleVal accessor", func(t *testing.T) {
		require.Equal(t, 2.0, float64Double().DoubleVal())
	})

	t.Run("FloatVal accessor wraps", func(t *testing.T) {
		fv := float64Double().FloatVal()
		require.NotNil(t, fv)
		require.Equal(t, 2.0, fv.Float64())
	})

	t.Run("string cast", func(t *testing.T) {
		s, err := atomicToString(float64Double())
		require.NoError(t, err)
		require.Equal(t, "2", s)
	})

	t.Run("distinct-values", func(t *testing.T) {
		seq := SingleAtomic(float64Double())
		got, err := fnDistinctValues(t.Context(), []Sequence{seq})
		require.NoError(t, err)
		require.Equal(t, 1, seqLen(got))
	})

	t.Run("max with FloatValue-backed double", func(t *testing.T) {
		arg := ItemSlice{float64Double(), AtomicValue{TypeName: TypeDouble, Value: NewDouble(1)}}
		got, err := fnMax(t.Context(), []Sequence{arg})
		require.NoError(t, err)
		require.Equal(t, 1, seqLen(got))
		av, ok := got.(ItemSlice)[0].(AtomicValue)
		require.True(t, ok)
		require.Equal(t, 2.0, av.ToFloat64())
	})

	t.Run("effective boolean value", func(t *testing.T) {
		b, err := ebvAtomic(float64Double())
		require.NoError(t, err)
		require.True(t, b)
	})
}

// TestDistinctValuesSchemaDerivedCollapse verifies that distinct-values folds a
// schema-derived value with its built-in equivalent via the BaseType-aware fast
// key, rather than keying solely on TypeName.
func TestDistinctValuesSchemaDerivedCollapse(t *testing.T) {
	t.Run("schema-derived NCName collapses with string", func(t *testing.T) {
		derived := AtomicValue{TypeName: "my:code", BaseType: TypeNCName, Value: "x"}
		plain := AtomicValue{TypeName: TypeString, Value: "x"}
		arg := ItemSlice{derived, plain}
		got, err := fnDistinctValues(t.Context(), []Sequence{arg})
		require.NoError(t, err)
		require.Equal(t, 1, seqLen(got))
	})

	t.Run("schema-derived decimal collapses with decimal", func(t *testing.T) {
		derived := AtomicValue{TypeName: "my:one", BaseType: TypeDecimal, Value: big.NewRat(1, 1)}
		plain := AtomicValue{TypeName: TypeDecimal, Value: big.NewRat(1, 1)}
		arg := ItemSlice{derived, plain}
		got, err := fnDistinctValues(t.Context(), []Sequence{arg})
		require.NoError(t, err)
		require.Equal(t, 1, seqLen(got))
	})
}

// TestDistinctValuesSchemaDerivedIntegerDiscriminates verifies the distinct-values
// fast key promotes via BaseType so two DIFFERENT schema-derived integers do NOT
// collapse to a single key. Before the fix the integer fast key passed the
// un-promoted value to toRatForCompare, which only recognizes built-in integer
// TypeNames and converted every custom-typed integer to 0 — so a custom-typed 1
// and 2 both keyed as 0 and distinct-values reported a single item.
func TestDistinctValuesSchemaDerivedIntegerDiscriminates(t *testing.T) {
	derivedInt := func(v any) AtomicValue {
		return AtomicValue{TypeName: "my:int", BaseType: TypeInteger, Value: v}
	}

	t.Run("int64-backed", func(t *testing.T) {
		arg := ItemSlice{derivedInt(int64(1)), derivedInt(int64(2))}
		got, err := fnDistinctValues(t.Context(), []Sequence{arg})
		require.NoError(t, err)
		require.Equal(t, 2, seqLen(got))
	})

	t.Run("big.Int-backed", func(t *testing.T) {
		arg := ItemSlice{derivedInt(big.NewInt(1)), derivedInt(big.NewInt(2))}
		got, err := fnDistinctValues(t.Context(), []Sequence{arg})
		require.NoError(t, err)
		require.Equal(t, 2, seqLen(got))
	})

	t.Run("collapses with built-in integer", func(t *testing.T) {
		plain := AtomicValue{TypeName: TypeInteger, Value: int64(1)}
		arg := ItemSlice{derivedInt(int64(1)), plain}
		got, err := fnDistinctValues(t.Context(), []Sequence{arg})
		require.NoError(t, err)
		require.Equal(t, 1, seqLen(got))
	})
}

// TestFloat64BackedSchemaDerived verifies the numeric accessors resolve the
// effective numeric type from BaseType for schema-derived float/double atomics
// (custom TypeName, built-in BaseType) backed by float64, float32, or
// *FloatValue. Before the fix ToFloat64 was TypeName-only and returned 0 for
// these, cascading into DoubleVal/FloatVal returning 0.
func TestFloat64BackedSchemaDerived(t *testing.T) {
	type backing struct {
		name  string
		value any
	}
	doubleBackings := []backing{
		{"float64", float64(2)},
		{"float32", float32(2)},
		{"FloatValue", NewDouble(2)},
	}
	floatBackings := []backing{
		{"float64", float64(2)},
		{"float32", float32(2)},
		{"FloatValue", NewFloat(2)},
	}

	t.Run("BaseType xs:double", func(t *testing.T) {
		for _, b := range doubleBackings {
			t.Run(b.name, func(t *testing.T) {
				a := AtomicValue{TypeName: "my:double", BaseType: TypeDouble, Value: b.value}
				require.Equal(t, 2.0, a.ToFloat64())
				require.Equal(t, 2.0, a.DoubleVal())
				fv := a.FloatVal()
				require.NotNil(t, fv)
				require.Equal(t, 2.0, fv.Float64())
				require.Equal(t, uint(PrecisionDouble), fv.Precision())
			})
		}
	})

	t.Run("BaseType xs:float", func(t *testing.T) {
		for _, b := range floatBackings {
			t.Run(b.name, func(t *testing.T) {
				a := AtomicValue{TypeName: "my:float", BaseType: TypeFloat, Value: b.value}
				require.Equal(t, 2.0, a.ToFloat64())
				require.Equal(t, 2.0, a.DoubleVal())
				fv := a.FloatVal()
				require.NotNil(t, fv)
				require.Equal(t, 2.0, fv.Float64())
				// xs:float base must yield single-precision.
				require.Equal(t, uint(PrecisionFloat), fv.Precision())
			})
		}
	})
}

// TestDistinctValuesSchemaDerivedDoubleDiscriminates verifies the distinct-values
// fast key promotes via BaseType so a schema-derived my:double(2) does NOT
// collapse with the built-in double 0e0: count(distinct-values(($d, 0e0))) == 2.
// Before the fix the fast key keyed both on the un-promoted ToFloat64() == 0 and
// reported a single distinct value.
func TestDistinctValuesSchemaDerivedDoubleDiscriminates(t *testing.T) {
	d := AtomicValue{TypeName: "my:double", BaseType: TypeDouble, Value: float64(2)}
	zero := AtomicValue{TypeName: TypeDouble, Value: NewDouble(0)}
	arg := ItemSlice{d, zero}
	got, err := fnDistinctValues(t.Context(), []Sequence{arg})
	require.NoError(t, err)
	require.Equal(t, 2, seqLen(got))
}

// TestDistinctValuesSchemaDerivedNaN verifies that a schema-derived NaN (custom
// TypeName, BaseType xs:double) is recognized as NaN and collapses with a
// built-in double NaN per op:is-same-key (all NaN are equal). Before the fix
// isAtomicNaN was TypeName-only, so the schema-derived NaN escaped the NaN
// branch and was reported as distinct.
func TestDistinctValuesSchemaDerivedNaN(t *testing.T) {
	derivedNaN := AtomicValue{TypeName: "my:double", BaseType: TypeDouble, Value: NewDouble(math.NaN())}
	builtinNaN := AtomicValue{TypeName: TypeDouble, Value: NewDouble(math.NaN())}
	arg := ItemSlice{derivedNaN, builtinNaN}
	got, err := fnDistinctValues(t.Context(), []Sequence{arg})
	require.NoError(t, err)
	require.Equal(t, 1, seqLen(got))
}

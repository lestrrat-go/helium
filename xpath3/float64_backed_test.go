package xpath3

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// tnMyFloat is a schema-derived xs:float type name reused across these tests.
const tnMyFloat = "my:float"

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

// TestFloatValSchemaDerivedFloatNarrows verifies FloatVal narrows a schema-derived
// xs:float to single precision even when the backing value is an existing
// double-precision *FloatValue. 16777217 is the smallest integer not exactly
// representable as float32; at single precision it rounds to 16777216. Before the
// fix FloatVal returned the existing *FloatValue unchanged, yielding the
// precision-53 value 16777217.
func TestFloatValSchemaDerivedFloatNarrows(t *testing.T) {
	a := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: NewDouble(16777217)}
	fv := a.FloatVal()
	require.NotNil(t, fv)
	require.Equal(t, uint(PrecisionFloat), fv.Precision())
	require.Equal(t, 16777216.0, fv.Float64())
}

// TestAggregateSchemaDerivedFloatWidth verifies fn:max/fn:min preserve xs:float
// width for a schema-derived xs:float operand. max(($x, xs:float("16777216")))
// with $x = my:float(16777217) must return an xs:float of 1.6777216E7, not an
// xs:double of 1.6777217E7. Before the fix promoteForAggregate ignored BaseType and
// promoted the float64-backed value to xs:double.
func TestAggregateSchemaDerivedFloatWidth(t *testing.T) {
	derivedFloat := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: NewDouble(16777217)}
	builtinFloat := AtomicValue{TypeName: TypeFloat, Value: NewFloat(16777216)}

	check := func(t *testing.T, got Sequence) {
		t.Helper()
		require.Equal(t, 1, seqLen(got))
		av, ok := got.(ItemSlice)[0].(AtomicValue)
		require.True(t, ok)
		require.Equal(t, TypeFloat, av.TypeName, "must preserve xs:float width")
		require.Equal(t, uint(PrecisionFloat), av.FloatVal().Precision())
		require.Equal(t, 16777216.0, av.ToFloat64())
	}

	t.Run("max", func(t *testing.T) {
		got, err := fnMax(t.Context(), []Sequence{ItemSlice{derivedFloat, builtinFloat}})
		require.NoError(t, err)
		check(t, got)
	})

	t.Run("min", func(t *testing.T) {
		got, err := fnMin(t.Context(), []Sequence{ItemSlice{derivedFloat, builtinFloat}})
		require.NoError(t, err)
		check(t, got)
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
				a := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: b.value}
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

// TestFloat64SchemaDerivedFloatNarrows verifies ToFloat64/DoubleVal narrow a
// schema-derived xs:float (custom TypeName, BaseType xs:float) to single
// precision for every backing form, matching FloatVal. 16777217 is the smallest
// integer not exactly representable as float32; at single precision it rounds to
// 16777216. Before the fix ToFloat64 returned the raw double value 16777217,
// leaking double precision into DoubleVal, mixed comparison, and duration math.
func TestFloat64SchemaDerivedFloatNarrows(t *testing.T) {
	const narrowed = 16777216.0

	backings := []struct {
		name  string
		value any
	}{
		{"float64", float64(16777217)},
		{"FloatValue-double", NewDouble(16777217)},
	}

	for _, b := range backings {
		t.Run(b.name, func(t *testing.T) {
			a := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: b.value}
			require.Equal(t, narrowed, a.ToFloat64())
			require.Equal(t, narrowed, a.DoubleVal())
			// ToFloat64 and FloatVal must agree for xs:float.
			require.Equal(t, a.ToFloat64(), a.FloatVal().Float64())
		})
	}

	t.Run("FloatValue-single already narrowed", func(t *testing.T) {
		// A *FloatValue already at single precision is unaffected.
		a := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: NewFloat(16777217)}
		require.Equal(t, narrowed, a.ToFloat64())
		require.Equal(t, narrowed, a.DoubleVal())
		require.Equal(t, a.ToFloat64(), a.FloatVal().Float64())
	})

	t.Run("built-in xs:double not narrowed", func(t *testing.T) {
		// The double path must be unaffected: 16777217 stays exact.
		a := AtomicValue{TypeName: TypeDouble, Value: NewDouble(16777217)}
		require.Equal(t, 16777217.0, a.ToFloat64())
		require.Equal(t, 16777217.0, a.DoubleVal())
	})
}

// TestDistinctValuesSchemaDerivedFloatNarrowingCollapse verifies the narrowing
// makes a schema-derived xs:float(16777217) compare equal to the built-in
// xs:float 16777216 via the value-space numeric key: both narrow to 16777216, so
// distinct-values folds them. Before the fix the schema-derived value keyed on
// the un-narrowed double 16777217 and was reported as distinct.
func TestDistinctValuesSchemaDerivedFloatNarrowingCollapse(t *testing.T) {
	derived := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: NewDouble(16777217)}
	builtin := AtomicValue{TypeName: TypeFloat, Value: NewFloat(16777216)}
	arg := ItemSlice{derived, builtin}
	got, err := fnDistinctValues(t.Context(), []Sequence{arg})
	require.NoError(t, err)
	require.Equal(t, 1, seqLen(got))
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

// TestEbvSchemaDerivedFloatNarrows verifies the effective-boolean-value path
// computes magnitude from the effective-typed value, not the raw float64 backing.
// A my:float (BaseType xs:float) backed by 1e-50 underflows to 0.0 at single
// precision, so its EBV must be false. Before the fix ebvAtomic tested the raw
// double 1e-50 (non-zero) and returned true. A built-in xs:double 1e-50 is NOT
// narrowed and stays true (control), and a NaN-backed xs:float yields false.
func TestEbvSchemaDerivedFloatNarrows(t *testing.T) {
	t.Run("my:float 1e-50 underflows to false", func(t *testing.T) {
		a := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: float64(1e-50)}
		b, err := ebvAtomic(a)
		require.NoError(t, err)
		require.False(t, b)
	})

	t.Run("my:float 2 is true", func(t *testing.T) {
		a := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: float64(2)}
		b, err := ebvAtomic(a)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("built-in xs:double 1e-50 stays true", func(t *testing.T) {
		a := AtomicValue{TypeName: TypeDouble, Value: NewDouble(1e-50)}
		b, err := ebvAtomic(a)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("NaN-backed my:float is false", func(t *testing.T) {
		a := AtomicValue{TypeName: tnMyFloat, BaseType: TypeFloat, Value: math.NaN()}
		b, err := ebvAtomic(a)
		require.NoError(t, err)
		require.False(t, b)
	})
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

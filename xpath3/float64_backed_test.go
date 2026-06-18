package xpath3

import (
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

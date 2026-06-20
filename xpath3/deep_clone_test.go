package xpath3_test

import (
	"math/big"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// XPath 3.1 maps and arrays are immutable / have value semantics: a value
// inserted into a map or array must be detached from the caller's copy so that a
// later mutation of the original (e.g. via a shared *big.Int pointer, or by
// mutating a nested map/array) cannot reach the stored member. These tests pin
// that the clone-on-insert is DEEP, not shallow.
func TestDeepCloneValueSemantics(t *testing.T) {
	t.Parallel()

	// bigIntSeq builds a single-item sequence wrapping a *big.Int-backed
	// xs:integer. The returned *big.Int is the shared backing pointer so the
	// caller can mutate it after insertion.
	bigIntSeq := func(n int64) (xpath3.Sequence, *big.Int) {
		bi := big.NewInt(n)
		return xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: bi}}, bi
	}

	readInt := func(t *testing.T, seq xpath3.Sequence) int64 {
		t.Helper()
		items := seq.Materialize()
		require.Len(t, items, 1)
		av, ok := items[0].(xpath3.AtomicValue)
		require.True(t, ok, "expected AtomicValue, got %T", items[0])
		return av.BigInt().Int64()
	}

	t.Run("array member with pointer-backed atomic", func(t *testing.T) {
		t.Parallel()

		seq, bi := bigIntSeq(7)
		arr := xpath3.NewArray([]xpath3.Sequence{seq})

		// Mutate the original *big.Int after it was inserted.
		bi.SetInt64(999)

		got, err := arr.Get(1)
		require.NoError(t, err)
		require.Equal(t, int64(7), readInt(t, got), "array member must be unaffected by mutation of the source pointer")
	})

	t.Run("map value with pointer-backed atomic", func(t *testing.T) {
		t.Parallel()

		seq, bi := bigIntSeq(7)
		key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
		m := xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: seq}})

		bi.SetInt64(999)

		got, ok := m.Get(key)
		require.True(t, ok)
		require.Equal(t, int64(7), readInt(t, got), "map value must be unaffected by mutation of the source pointer")
	})

	t.Run("map:put detaches inserted pointer-backed atomic", func(t *testing.T) {
		t.Parallel()

		base := xpath3.NewMap([]xpath3.MapEntry{
			{Key: xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "x"}, Value: xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(1)}}},
			{Key: xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "y"}, Value: xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(2)}}},
		})

		seq, bi := bigIntSeq(7)
		key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "z"}
		m2 := base.Put(key, seq)

		bi.SetInt64(999)

		got, ok := m2.Get(key)
		require.True(t, ok)
		require.Equal(t, int64(7), readInt(t, got))
	})

	t.Run("nested array shared backing is detached", func(t *testing.T) {
		t.Parallel()

		innerSeq, bi := bigIntSeq(7)
		inner := xpath3.NewArray([]xpath3.Sequence{innerSeq})

		// Insert the inner array as a member of an outer array. The outer array's
		// stored member must not share the inner array's pointer-backed atomic.
		outer := xpath3.NewArray([]xpath3.Sequence{xpath3.ItemSlice{inner}})

		bi.SetInt64(999)

		got, err := outer.Get(1)
		require.NoError(t, err)
		items := got.Materialize()
		require.Len(t, items, 1)
		gotInner, ok := items[0].(xpath3.ArrayItem)
		require.True(t, ok, "expected nested ArrayItem, got %T", items[0])
		innerMember, err := gotInner.Get(1)
		require.NoError(t, err)
		require.Equal(t, int64(7), readInt(t, innerMember), "nested array member must be detached from the source pointer")
	})

	t.Run("nested map inside array is detached", func(t *testing.T) {
		t.Parallel()

		key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
		valSeq, bi := bigIntSeq(7)
		inner := xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: valSeq}})

		arr := xpath3.NewArray([]xpath3.Sequence{xpath3.ItemSlice{inner}})

		bi.SetInt64(999)

		got, err := arr.Get(1)
		require.NoError(t, err)
		items := got.Materialize()
		require.Len(t, items, 1)
		gotMap, ok := items[0].(xpath3.MapItem)
		require.True(t, ok, "expected nested MapItem, got %T", items[0])
		gotVal, ok := gotMap.Get(key)
		require.True(t, ok)
		require.Equal(t, int64(7), readInt(t, gotVal), "nested map value must be detached from the source pointer")
	})

	t.Run("byte-slice atomic is detached", func(t *testing.T) {
		t.Parallel()

		buf := []byte{0x01, 0x02, 0x03}
		seq := xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeHexBinary, Value: buf}}
		arr := xpath3.NewArray([]xpath3.Sequence{seq})

		// Mutate the original byte slice in place.
		buf[0] = 0xff

		got, err := arr.Get(1)
		require.NoError(t, err)
		items := got.Materialize()
		require.Len(t, items, 1)
		av, ok := items[0].(xpath3.AtomicValue)
		require.True(t, ok)
		stored, ok := av.Value.([]byte)
		require.True(t, ok, "expected []byte, got %T", av.Value)
		require.Equal(t, []byte{0x01, 0x02, 0x03}, stored, "stored byte slice must be detached from the source slice")
	})
}

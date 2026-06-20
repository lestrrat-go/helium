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

	// bigIntKey builds a *big.Int-backed xs:integer AtomicValue suitable for use
	// as a map key, returning the shared backing pointer so the caller can mutate
	// it after insertion.
	bigIntKey := func(n int64) (xpath3.AtomicValue, *big.Int) {
		bi := big.NewInt(n)
		return xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: bi}, bi
	}

	keyInt := func(t *testing.T, k xpath3.AtomicValue) int64 {
		t.Helper()
		return k.BigInt().Int64()
	}

	t.Run("NewMap detaches pointer-backed key", func(t *testing.T) {
		t.Parallel()

		key, bi := bigIntKey(7)
		val := xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "v"}}
		m := xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: val}})

		// Mutate the original *big.Int after it was inserted as a key.
		bi.SetInt64(999)

		// The stored key must still be 7: lookup with a fresh 7 key must hit, and
		// the key returned by Keys must read 7.
		lookup, _ := bigIntKey(7)
		_, ok := m.Get(lookup)
		require.True(t, ok, "stored key must be unaffected by mutation of the source pointer")

		keys := m.Keys()
		require.Len(t, keys, 1)
		require.Equal(t, int64(7), keyInt(t, keys[0]), "Keys must return the detached stored key")
	})

	t.Run("map:put detaches pointer-backed key", func(t *testing.T) {
		t.Parallel()

		base := xpath3.NewMap([]xpath3.MapEntry{
			{Key: xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "x"}, Value: xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(1)}}},
		})

		key, bi := bigIntKey(7)
		val := xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "v"}}
		m2 := base.Put(key, val)

		bi.SetInt64(999)

		lookup, _ := bigIntKey(7)
		_, ok := m2.Get(lookup)
		require.True(t, ok, "Put-stored key must be unaffected by mutation of the source pointer")
	})

	t.Run("ForEach callback cannot mutate stored map", func(t *testing.T) {
		t.Parallel()

		key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
		seq, _ := bigIntSeq(7)
		m := xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: seq}})

		// Mutate the value's pointer-backed atomic through the ForEach callback.
		err := m.ForEach(func(_ xpath3.AtomicValue, v xpath3.Sequence) error {
			items := v.Materialize()
			require.Len(t, items, 1)
			av, ok := items[0].(xpath3.AtomicValue)
			require.True(t, ok)
			bi, ok := av.Value.(*big.Int)
			require.True(t, ok, "expected *big.Int, got %T", av.Value)
			bi.SetInt64(999)
			return nil
		})
		require.NoError(t, err)

		got, ok := m.Get(key)
		require.True(t, ok)
		require.Equal(t, int64(7), readInt(t, got), "map value must be unaffected by mutation through the ForEach callback")
	})

	t.Run("Flatten detaches pointer-backed atomic", func(t *testing.T) {
		t.Parallel()

		seq := xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(7)}}
		arr := xpath3.NewArray([]xpath3.Sequence{seq})

		flat := arr.Flatten().Materialize()
		require.Len(t, flat, 1)
		av, ok := flat[0].(xpath3.AtomicValue)
		require.True(t, ok, "expected AtomicValue, got %T", flat[0])
		bi, ok := av.Value.(*big.Int)
		require.True(t, ok, "expected *big.Int, got %T", av.Value)

		// Mutate the *big.Int obtained from Flatten; the original array's stored
		// member must be unaffected.
		bi.SetInt64(999)

		got, err := arr.Get(1)
		require.NoError(t, err)
		require.Equal(t, int64(7), readInt(t, got), "array member must be unaffected by mutation of a value obtained from Flatten")
	})

	// mutateBigInt finds the single *big.Int-backed atomic in seq and overwrites
	// it in place, simulating a caller mutating a value it got back from a lookup.
	mutateBigInt := func(t *testing.T, seq xpath3.Sequence) {
		t.Helper()
		items := seq.Materialize()
		require.Len(t, items, 1)
		av, ok := items[0].(xpath3.AtomicValue)
		require.True(t, ok, "expected AtomicValue, got %T", items[0])
		bi, ok := av.Value.(*big.Int)
		require.True(t, ok, "expected *big.Int, got %T", av.Value)
		bi.SetInt64(999)
	}

	// The `?*` and keyed `?key` lookup paths, and map:get, must hand back a
	// defensive clone of the borrowed stored value — not the map's own backing
	// sequence. Under EvalBorrowing the variable map is the same Go MapItem we
	// hold here, so a regression that returns the stored value lets a mutation of
	// the lookup output reach the source map. Each case mutates the lookup result
	// and asserts the source map still reads its original value.
	for _, tc := range []struct {
		name string
		expr string
	}{
		{name: "wildcard lookup output is detached", expr: `$m ! ?*`},
		{name: "keyed lookup output is detached", expr: `$m ! ?k`},
		{name: "map:get output is detached", expr: `map:get($m, "k")`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "k"}
			seq, _ := bigIntSeq(7)
			m := xpath3.NewMap([]xpath3.MapEntry{{Key: key, Value: seq}})

			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)

			result, err := xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(varsSet("m", xpath3.ItemSlice{m})).
				Evaluate(t.Context(), compiled, nil)
			require.NoError(t, err)

			// Mutate the *big.Int returned by the lookup.
			mutateBigInt(t, result.Sequence())

			got, ok := m.Get(key)
			require.True(t, ok)
			require.Equal(t, int64(7), readInt(t, got), "source map value must be unaffected by mutation of the lookup output")
		})
	}

	// The array `?*` wildcard and keyed `?n` lookup paths must hand back a
	// defensive clone of the borrowed stored member — not the array's own backing
	// sequence. Under EvalBorrowing the variable array is the same Go ArrayItem we
	// hold here, so a regression that returns the stored member lets a mutation of
	// the lookup output reach the source array. Each case mutates the lookup
	// result and asserts the source array still reads its original value.
	for _, tc := range []struct {
		name string
		expr string
	}{
		{name: "array wildcard lookup output is detached", expr: `$a ! ?*`},
		{name: "array keyed lookup output is detached", expr: `$a ! ?1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seq, _ := bigIntSeq(7)
			arr := xpath3.NewArray([]xpath3.Sequence{seq})

			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)

			result, err := xpath3.NewEvaluator(xpath3.EvalBorrowing).
				Variables(varsSet("a", xpath3.ItemSlice{arr})).
				Evaluate(t.Context(), compiled, nil)
			require.NoError(t, err)

			// Mutate the *big.Int returned by the lookup.
			mutateBigInt(t, result.Sequence())

			got, err := arr.Get(1)
			require.NoError(t, err)
			require.Equal(t, int64(7), readInt(t, got), "source array member must be unaffected by mutation of the lookup output")
		})
	}

	// The array:flatten() function (not just the ArrayItem.Flatten() method)
	// must hand back deep-cloned leaf items. Under EvalBorrowing the variable
	// array is the same Go ArrayItem we hold here, so a regression that appends
	// the borrowed stored item lets a mutation of the flattened output reach the
	// source array. Mutate the result and assert the source array is unchanged.
	t.Run("array:flatten output is detached", func(t *testing.T) {
		t.Parallel()

		seq, _ := bigIntSeq(7)
		arr := xpath3.NewArray([]xpath3.Sequence{seq})

		compiled, err := xpath3.NewCompiler().Compile(`array:flatten($a)`)
		require.NoError(t, err)

		result, err := xpath3.NewEvaluator(xpath3.EvalBorrowing).
			Variables(varsSet("a", xpath3.ItemSlice{arr})).
			Evaluate(t.Context(), compiled, nil)
		require.NoError(t, err)

		// Mutate the *big.Int returned by flatten.
		mutateBigInt(t, result.Sequence())

		got, err := arr.Get(1)
		require.NoError(t, err)
		require.Equal(t, int64(7), readInt(t, got), "source array member must be unaffected by mutation of the array:flatten output")
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

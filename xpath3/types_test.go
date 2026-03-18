package xpath3_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestAtomicValueAccessors(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "hello"}
		require.Equal(t, "hello", v.StringVal())
		require.False(t, v.IsNumeric())
	})

	t.Run("integer", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(42)}
		require.Equal(t, int64(42), v.IntegerVal())
		require.True(t, v.IsNumeric())
		require.InDelta(t, float64(42), v.ToFloat64(), 0)
	})

	t.Run("double", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(3.14)}
		require.InDelta(t, 3.14, v.DoubleVal(), 1e-9)
		require.True(t, v.IsNumeric())
	})

	t.Run("boolean", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeBoolean, Value: true}
		require.True(t, v.BooleanVal())
	})

	t.Run("schema-derived fallback atomization", func(t *testing.T) {
		doc, err := helium.Parse(t.Context(), []byte(`<root>-0</root>`))
		require.NoError(t, err)

		root := doc.FirstChild()
		require.NotNil(t, root)

		av, err := xpath3.AtomizeItem(xpath3.NodeItem{
			Node:           root,
			TypeAnnotation: "Q{urn:test}derived-float",
			AtomizedType:   xpath3.TypeFloat,
		})
		require.NoError(t, err)
		require.Equal(t, xpath3.TypeFloat, av.TypeName)

		s, err := xpath3.AtomicToString(av)
		require.NoError(t, err)
		require.Equal(t, "-0", s)
	})
}

func TestMapItem(t *testing.T) {
	strKey := func(s string) xpath3.AtomicValue {
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}
	}

	t.Run("basic operations", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: xpath3.SingleInteger(1)},
			{Key: strKey("b"), Value: xpath3.SingleInteger(2)},
		})

		require.Equal(t, 2, m.Size())
		require.True(t, m.Contains(strKey("a")))
		require.False(t, m.Contains(strKey("c")))

		v, ok := m.Get(strKey("a"))
		require.True(t, ok)
		require.Len(t, v, 1)
		av := v[0].(xpath3.AtomicValue)
		require.Equal(t, int64(1), av.IntegerVal())
	})

	t.Run("keys order", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("z"), Value: xpath3.SingleInteger(1)},
			{Key: strKey("a"), Value: xpath3.SingleInteger(2)},
			{Key: strKey("m"), Value: xpath3.SingleInteger(3)},
		})
		keys := m.Keys()
		require.Len(t, keys, 3)
		require.Equal(t, "z", keys[0].StringVal())
		require.Equal(t, "a", keys[1].StringVal())
		require.Equal(t, "m", keys[2].StringVal())
	})

	t.Run("put new key", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: xpath3.SingleInteger(1)},
		})
		m2 := m.Put(strKey("b"), xpath3.SingleInteger(2))
		require.Equal(t, 1, m.Size())  // original unchanged
		require.Equal(t, 2, m2.Size()) // new has both
	})

	t.Run("put replace", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: xpath3.SingleInteger(1)},
		})
		m2 := m.Put(strKey("a"), xpath3.SingleInteger(99))
		require.Equal(t, 1, m2.Size())
		v, ok := m2.Get(strKey("a"))
		require.True(t, ok)
		require.Equal(t, int64(99), v[0].(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("remove", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: xpath3.SingleInteger(1)},
			{Key: strKey("b"), Value: xpath3.SingleInteger(2)},
		})
		m2 := m.Remove(strKey("a"))
		require.Equal(t, 1, m2.Size())
		require.False(t, m2.Contains(strKey("a")))
		require.True(t, m2.Contains(strKey("b")))
	})

	t.Run("foreach", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: xpath3.SingleInteger(1)},
			{Key: strKey("b"), Value: xpath3.SingleInteger(2)},
		})
		var keys []string
		err := m.ForEach(func(k xpath3.AtomicValue, _ xpath3.Sequence) error {
			keys = append(keys, k.StringVal())
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, []string{"a", "b"}, keys)
	})

	t.Run("time key normalization", func(t *testing.T) {
		t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 1, 13, 0, 0, 0, time.FixedZone("UTC+1", 3600))
		k1 := xpath3.AtomicValue{TypeName: xpath3.TypeDateTime, Value: t1}
		k2 := xpath3.AtomicValue{TypeName: xpath3.TypeDateTime, Value: t2}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: k1, Value: xpath3.SingleString("found")},
		})
		// t2 is the same instant as t1 in UTC, so lookup should succeed
		v, ok := m.Get(k2)
		require.True(t, ok)
		require.Equal(t, "found", v[0].(xpath3.AtomicValue).StringVal())
	})

	t.Run("constructor clones value sequences", func(t *testing.T) {
		value := xpath3.SingleString("original")
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: value},
		})

		value[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "mutated"}

		got, ok := m.Get(strKey("a"))
		require.True(t, ok)
		require.Equal(t, "original", got[0].(xpath3.AtomicValue).StringVal())
	})

	t.Run("get returns cloned value sequence", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: xpath3.SingleString("original")},
		})

		got, ok := m.Get(strKey("a"))
		require.True(t, ok)
		got[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "mutated"}

		again, ok := m.Get(strKey("a"))
		require.True(t, ok)
		require.Equal(t, "original", again[0].(xpath3.AtomicValue).StringVal())
	})
}

func TestMergeMaps(t *testing.T) {
	strKey := func(s string) xpath3.AtomicValue {
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}
	}

	m1 := xpath3.NewMap([]xpath3.MapEntry{
		{Key: strKey("a"), Value: xpath3.SingleInteger(1)},
		{Key: strKey("b"), Value: xpath3.SingleInteger(2)},
	})
	m2 := xpath3.NewMap([]xpath3.MapEntry{
		{Key: strKey("b"), Value: xpath3.SingleInteger(20)},
		{Key: strKey("c"), Value: xpath3.SingleInteger(3)},
	})

	t.Run("use first", func(t *testing.T) {
		merged, err := xpath3.MergeMaps([]xpath3.MapItem{m1, m2}, xpath3.MergeUseFirst)
		require.NoError(t, err)
		require.Equal(t, 3, merged.Size())
		v, _ := merged.Get(strKey("b"))
		require.Equal(t, int64(2), v[0].(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("use last", func(t *testing.T) {
		merged, err := xpath3.MergeMaps([]xpath3.MapItem{m1, m2}, xpath3.MergeUseLast)
		require.NoError(t, err)
		v, _ := merged.Get(strKey("b"))
		require.Equal(t, int64(20), v[0].(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("reject", func(t *testing.T) {
		_, err := xpath3.MergeMaps([]xpath3.MapItem{m1, m2}, xpath3.MergeReject)
		require.Error(t, err)
	})
}

func TestArrayItem(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{
			xpath3.SingleInteger(10),
			xpath3.SingleInteger(20),
			xpath3.SingleInteger(30),
		})
		require.Equal(t, 3, a.Size())

		v, err := a.Get(1)
		require.NoError(t, err)
		require.Equal(t, int64(10), v[0].(xpath3.AtomicValue).IntegerVal())

		v, err = a.Get(3)
		require.NoError(t, err)
		require.Equal(t, int64(30), v[0].(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("out of bounds", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{xpath3.SingleInteger(1)})
		_, err := a.Get(0)
		require.Error(t, err)
		_, err = a.Get(2)
		require.Error(t, err)
	})

	t.Run("put", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{
			xpath3.SingleInteger(1),
			xpath3.SingleInteger(2),
		})
		a2, err := a.Put(2, xpath3.SingleInteger(99))
		require.NoError(t, err)
		v, _ := a2.Get(2)
		require.Equal(t, int64(99), v[0].(xpath3.AtomicValue).IntegerVal())
		// Original unchanged
		v, _ = a.Get(2)
		require.Equal(t, int64(2), v[0].(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("append", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{xpath3.SingleInteger(1)})
		a2 := a.Append(xpath3.SingleInteger(2))
		require.Equal(t, 1, a.Size())
		require.Equal(t, 2, a2.Size())
	})

	t.Run("subarray", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{
			xpath3.SingleInteger(10),
			xpath3.SingleInteger(20),
			xpath3.SingleInteger(30),
			xpath3.SingleInteger(40),
		})
		sub, err := a.SubArray(2, 2)
		require.NoError(t, err)
		require.Equal(t, 2, sub.Size())
		v, _ := sub.Get(1)
		require.Equal(t, int64(20), v[0].(xpath3.AtomicValue).IntegerVal())
		v, _ = sub.Get(2)
		require.Equal(t, int64(30), v[0].(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("flatten", func(t *testing.T) {
		inner := xpath3.NewArray([]xpath3.Sequence{
			xpath3.SingleInteger(3),
			xpath3.SingleInteger(4),
		})
		a := xpath3.NewArray([]xpath3.Sequence{
			xpath3.SingleInteger(1),
			xpath3.SingleInteger(2),
			{inner},
		})
		flat := a.Flatten()
		require.Len(t, flat, 4)
		require.Equal(t, int64(1), flat[0].(xpath3.AtomicValue).IntegerVal())
		require.Equal(t, int64(4), flat[3].(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("constructor clones member sequences", func(t *testing.T) {
		member := xpath3.SingleString("original")
		a := xpath3.NewArray([]xpath3.Sequence{member})

		member[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "mutated"}

		got, err := a.Get(1)
		require.NoError(t, err)
		require.Equal(t, "original", got[0].(xpath3.AtomicValue).StringVal())
	})

	t.Run("get returns cloned member sequence", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{xpath3.SingleString("original")})

		got, err := a.Get(1)
		require.NoError(t, err)
		got[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "mutated"}

		again, err := a.Get(1)
		require.NoError(t, err)
		require.Equal(t, "original", again[0].(xpath3.AtomicValue).StringVal())
	})

	t.Run("members returns cloned sequences", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{xpath3.SingleString("original")})

		members := a.Members()
		members[0][0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "mutated"}

		again, err := a.Get(1)
		require.NoError(t, err)
		require.Equal(t, "original", again[0].(xpath3.AtomicValue).StringVal())
	})
}

func TestSequenceHelpers(t *testing.T) {
	t.Run("empty sequence", func(t *testing.T) {
		seq := xpath3.EmptySequence()
		require.Len(t, seq, 0)
	})

	t.Run("single constructors", func(t *testing.T) {
		require.Len(t, xpath3.SingleBoolean(true), 1)
		require.Len(t, xpath3.SingleInteger(42), 1)
		require.Len(t, xpath3.SingleDouble(3.14), 1)
		require.Len(t, xpath3.SingleString("hi"), 1)
	})
}

func TestEBV(t *testing.T) {
	tests := []struct {
		name   string
		seq    xpath3.Sequence
		expect bool
		err    bool
	}{
		{"empty", xpath3.EmptySequence(), false, false},
		{"true", xpath3.SingleBoolean(true), true, false},
		{"false", xpath3.SingleBoolean(false), false, false},
		{"nonempty string", xpath3.SingleString("x"), true, false},
		{"empty string", xpath3.SingleString(""), false, false},
		{"nonzero integer", xpath3.SingleInteger(42), true, false},
		{"zero integer", xpath3.SingleInteger(0), false, false},
		{"nonzero double", xpath3.SingleDouble(1.5), true, false},
		{"zero double", xpath3.SingleDouble(0), false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := xpath3.EBV(tc.seq)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expect, result)
			}
		})
	}
}

func TestAtomizeSequence(t *testing.T) {
	seq := xpath3.Sequence{
		xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "hello"},
		xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(42)},
	}
	atoms, err := xpath3.AtomizeSequence(seq)
	require.NoError(t, err)
	require.Len(t, atoms, 2)
	require.Equal(t, "hello", atoms[0].StringVal())
	require.Equal(t, int64(42), atoms[1].IntegerVal())
}

func TestAtomizeFunction(t *testing.T) {
	seq := xpath3.Sequence{xpath3.FunctionItem{Arity: 0, Name: "test"}}
	_, err := xpath3.AtomizeSequence(seq)
	require.Error(t, err)
}

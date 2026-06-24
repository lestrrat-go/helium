package xpath3_test

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestAtomicValueAccessors(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testHello}
		require.Equal(t, testHello, v.StringVal())
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
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>-0</root>`))
		require.NoError(t, err)

		root := doc.FirstChild()
		require.NotNil(t, root)

		av, err := xpath3.AtomizeItem(xpath3.NodeItem{
			Node:           root,
			TypeAnnotation: "Q{urn:test}derived-float",
			AtomizedType:   xpath3.TypeFloat,
		})
		require.NoError(t, err)
		// Atomization preserves the user-defined type annotation so that
		// "instance of" checks match the original schema type.
		require.Equal(t, "Q{urn:test}derived-float", av.TypeName)

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
		av := v.Get(0).(xpath3.AtomicValue)
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
		require.Equal(t, int64(99), v.Get(0).(xpath3.AtomicValue).IntegerVal())
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
		require.Equal(t, "found", v.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("constructor clones value sequences", func(t *testing.T) {
		value := xpath3.SingleString("original")
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: value},
		})

		value.(xpath3.ItemSlice)[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testMutated}

		got, ok := m.Get(strKey("a"))
		require.True(t, ok)
		require.Equal(t, "original", got.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("get returns cloned value sequence", func(t *testing.T) {
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKey("a"), Value: xpath3.SingleString("original")},
		})

		got, ok := m.Get(strKey("a"))
		require.True(t, ok)
		got.(xpath3.ItemSlice)[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testMutated}

		again, ok := m.Get(strKey("a"))
		require.True(t, ok)
		require.Equal(t, "original", again.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("string-derived key matches string key", func(t *testing.T) {
		ncKey := xpath3.AtomicValue{TypeName: xpath3.TypeNCName, Value: "a"}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: ncKey, Value: xpath3.SingleInteger(1)},
		})
		// Looking up with an xs:string key of equal value space must succeed.
		v, ok := m.Get(strKey("a"))
		require.True(t, ok)
		require.Equal(t, int64(1), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("string and string-derived keys are equivalent", func(t *testing.T) {
		ncKey := xpath3.AtomicValue{TypeName: xpath3.TypeNCName, Value: "a"}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: ncKey, Value: xpath3.SingleInteger(1)},
			{Key: strKey("b"), Value: xpath3.SingleInteger(2)},
		})
		// xs:string("a") and xs:NCName("a") share a value-space key, so a
		// string lookup and Contains both resolve the NCName entry.
		require.True(t, m.Contains(strKey("a")))
		v, ok := m.Get(strKey("a"))
		require.True(t, ok)
		require.Equal(t, int64(1), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("duration key lookup", func(t *testing.T) {
		k1, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		k2, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: k1, Value: xpath3.SingleString("found")},
		})
		// k1 and k2 are independently parsed; their FracSec pointers differ
		// but the value space is identical, so lookup must succeed.
		v, ok := m.Get(k2)
		require.True(t, ok)
		require.Equal(t, "found", v.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("equivalent duration keys are not distinct", func(t *testing.T) {
		k1, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		k2, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		k3, err := xpath3.CastFromString("PT2S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: k1, Value: xpath3.SingleInteger(1)},
			{Key: k3, Value: xpath3.SingleInteger(3)},
		})
		// Independently parsed PT1.5S keys (differing FracSec pointers) collide.
		require.True(t, m.Contains(k2))
		// A distinct duration value must not collide.
		v, ok := m.Get(k3)
		require.True(t, ok)
		require.Equal(t, int64(3), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("arithmetic fractional duration matches parsed", func(t *testing.T) {
		parsed, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		// Arithmetic-created Duration carries the fraction in Seconds with a nil
		// FracSec; it must canonicalize to the same key as the parsed PT1.5S.
		arith := xpath3.AtomicValue{
			TypeName: xpath3.TypeDayTimeDuration,
			Value:    xpath3.Duration{Seconds: 1.5},
		}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: parsed, Value: xpath3.SingleString("found")},
		})
		v, ok := m.Get(arith)
		require.True(t, ok)
		require.Equal(t, "found", v.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("non-binary fractional duration via parsing matches arithmetic seconds", func(t *testing.T) {
		// PT0.1S is NOT binary-exact as a float64, so this case exercises the
		// exact-rational fractional-seconds canonicalization (unlike PT1.5S).
		parsed, err := xpath3.CastFromString("PT0.1S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		// Emulate an arithmetic-created duration that stores the fraction as an
		// exact FracSec rational of 1/10.
		arith := xpath3.AtomicValue{
			TypeName: xpath3.TypeDayTimeDuration,
			Value: xpath3.Duration{
				Seconds: 0.1,
				FracSec: big.NewRat(1, 10),
			},
		}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: parsed, Value: xpath3.SingleString("found")},
		})
		v, ok := m.Get(arith)
		require.True(t, ok)
		require.Equal(t, "found", v.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("equivalent duration keys collapse in NewMap index", func(t *testing.T) {
		parsed, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		arith := xpath3.AtomicValue{
			TypeName: xpath3.TypeDayTimeDuration,
			Value:    xpath3.Duration{Seconds: 1.5},
		}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: parsed, Value: xpath3.SingleInteger(1)},
			{Key: arith, Value: xpath3.SingleInteger(2)},
		})
		// NewMap is the low-level constructor and does NOT enforce XPath map
		// duplicate-key semantics (XQDY0137); that is enforced by the map
		// constructor expression before NewMap is reached. Here both entries
		// share a value-space key: NewMap retains both rows (Size stays 2) but
		// its lookup index collapses them so the last-indexed entry wins for
		// both keys.
		require.Equal(t, 2, m.Size())
		v, ok := m.Get(parsed)
		require.True(t, ok)
		require.Equal(t, int64(2), v.Get(0).(xpath3.AtomicValue).IntegerVal())
		v, ok = m.Get(arith)
		require.True(t, ok)
		require.Equal(t, int64(2), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("distinct fractional durations are not equal", func(t *testing.T) {
		k12, err := xpath3.CastFromString("PT1.2S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		k19, err := xpath3.CastFromString("PT1.9S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: k12, Value: xpath3.SingleInteger(12)},
			{Key: k19, Value: xpath3.SingleInteger(19)},
		})
		// 1.2S and 1.9S truncate to the same whole second but are distinct values.
		require.Equal(t, 2, m.Size())
		v12, ok := m.Get(k12)
		require.True(t, ok)
		require.Equal(t, int64(12), v12.Get(0).(xpath3.AtomicValue).IntegerVal())
		v19, ok := m.Get(k19)
		require.True(t, ok)
		require.Equal(t, int64(19), v19.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("negative zero duration matches positive zero", func(t *testing.T) {
		pz, err := xpath3.CastFromString("PT0S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		nz, err := xpath3.CastFromString("-PT0S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: pz, Value: xpath3.SingleString("zero")},
		})
		// Duration comparison treats -PT0S and PT0S as equal, so they share a key.
		v, ok := m.Get(nz)
		require.True(t, ok)
		require.Equal(t, "zero", v.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("schema-derived string atomic matches string key", func(t *testing.T) {
		// A schema-derived atomic carries a custom TypeName whose BaseType is the
		// string-like ancestor (here xs:NCName). It must fold against an equal
		// xs:string key via BaseType, not just TypeName.
		derived := xpath3.AtomicValue{
			TypeName: "Q{urn:test}myToken",
			BaseType: xpath3.TypeNCName,
			Value:    "tok",
		}
		strKeyVal := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "tok"}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: strKeyVal, Value: xpath3.SingleInteger(7)},
		})
		v, ok := m.Get(derived)
		require.True(t, ok)
		require.Equal(t, int64(7), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("schema-derived duration key matches built-in duration", func(t *testing.T) {
		// A schema-derived atomic carries a custom TypeName whose BaseType is a
		// built-in duration ancestor. It must fold to xs:duration via BaseType so
		// it shares a key with an equal built-in dayTimeDuration value.
		builtin, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		derivedVal, err := xpath3.CastFromString("PT1.5S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		derived := xpath3.AtomicValue{
			TypeName: "Q{urn:test}myDTD",
			BaseType: xpath3.TypeDayTimeDuration,
			Value:    derivedVal.Value,
		}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: builtin, Value: xpath3.SingleInteger(11)},
		})
		v, ok := m.Get(derived)
		require.True(t, ok)
		require.Equal(t, int64(11), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("built-in duration key matches schema-derived duration", func(t *testing.T) {
		// The reverse direction: a schema-derived duration is the stored key and a
		// built-in duration is the lookup key. Both must fold to xs:duration.
		derivedVal, err := xpath3.CastFromString("P1Y2M", xpath3.TypeYearMonthDuration)
		require.NoError(t, err)
		derived := xpath3.AtomicValue{
			TypeName: "Q{urn:test}myYMD",
			BaseType: xpath3.TypeYearMonthDuration,
			Value:    derivedVal.Value,
		}
		builtin, err := xpath3.CastFromString("P1Y2M", xpath3.TypeYearMonthDuration)
		require.NoError(t, err)
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: derived, Value: xpath3.SingleInteger(13)},
		})
		v, ok := m.Get(builtin)
		require.True(t, ok)
		require.Equal(t, int64(13), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("huge whole-second durations stay distinct map keys", func(t *testing.T) {
		// 2^53 and 2^53+1 are consecutive whole seconds whose float64
		// representations collapse to the same value. Storing dayTime seconds as
		// an exact rational keeps the two keys distinct.
		k0, err := xpath3.CastFromString("PT9007199254740992S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		k1, err := xpath3.CastFromString("PT9007199254740993S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)

		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: k0, Value: xpath3.SingleInteger(992)},
			{Key: k1, Value: xpath3.SingleInteger(993)},
		})
		require.Equal(t, 2, m.Size())

		// Each huge key resolves to its own value and matches a freshly re-parsed
		// equivalent of itself.
		reparsed0, err := xpath3.CastFromString("PT9007199254740992S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)
		reparsed1, err := xpath3.CastFromString("PT9007199254740993S", xpath3.TypeDayTimeDuration)
		require.NoError(t, err)

		v0, ok := m.Get(reparsed0)
		require.True(t, ok)
		require.Equal(t, int64(992), v0.Get(0).(xpath3.AtomicValue).IntegerVal())

		v1, ok := m.Get(reparsed1)
		require.True(t, ok)
		require.Equal(t, int64(993), v1.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("schema-derived double keys are distinct and do not match zero", func(t *testing.T) {
		// A schema-derived atomic carries a custom TypeName whose BaseType is
		// xs:double. The float map-key path must promote via BaseType before
		// ToFloat64, otherwise the underlying FloatValue is read as 0 and every
		// such key collapses to the same map slot.
		two := xpath3.AtomicValue{
			TypeName: "Q{urn:test}myDouble",
			BaseType: xpath3.TypeDouble,
			Value:    xpath3.NewDouble(2),
		}
		three := xpath3.AtomicValue{
			TypeName: "Q{urn:test}myDouble",
			BaseType: xpath3.TypeDouble,
			Value:    xpath3.NewDouble(3),
		}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: two, Value: xpath3.SingleString("two")},
			{Key: three, Value: xpath3.SingleString("three")},
		})
		// Two distinct derived-double keys must remain distinct, not collapse.
		require.Equal(t, 2, m.Size())

		v2, ok := m.Get(two)
		require.True(t, ok)
		require.Equal(t, "two", v2.Get(0).(xpath3.AtomicValue).StringVal())

		v3, ok := m.Get(three)
		require.True(t, ok)
		require.Equal(t, "three", v3.Get(0).(xpath3.AtomicValue).StringVal())

		// An integer 0 lookup must match neither derived-double key.
		_, ok = m.Get(xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(0)})
		require.False(t, ok)
	})

	t.Run("schema-derived float keys are distinct and do not match zero", func(t *testing.T) {
		two := xpath3.AtomicValue{
			TypeName: "Q{urn:test}myFloat",
			BaseType: xpath3.TypeFloat,
			Value:    xpath3.NewDouble(2),
		}
		three := xpath3.AtomicValue{
			TypeName: "Q{urn:test}myFloat",
			BaseType: xpath3.TypeFloat,
			Value:    xpath3.NewDouble(3),
		}
		m := xpath3.NewMap([]xpath3.MapEntry{
			{Key: two, Value: xpath3.SingleString("two")},
			{Key: three, Value: xpath3.SingleString("three")},
		})
		require.Equal(t, 2, m.Size())

		v2, ok := m.Get(two)
		require.True(t, ok)
		require.Equal(t, "two", v2.Get(0).(xpath3.AtomicValue).StringVal())

		v3, ok := m.Get(three)
		require.True(t, ok)
		require.Equal(t, "three", v3.Get(0).(xpath3.AtomicValue).StringVal())

		_, ok = m.Get(xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(0)})
		require.False(t, ok)
	})

	// 2^24 and 2^24+1 are distinct in float64/double but collapse to the same
	// value in IEEE-754 single precision (xs:float). The map-key path must round
	// schema-derived xs:float keys to single precision regardless of backing
	// (float64/float32/*FloatValue), or a lookup by one would miss the other.
	t.Run("schema-derived float keys round to single precision", func(t *testing.T) {
		const exact = 16777216   // 2^24, exact in float32
		const plusOne = 16777217 // 2^24+1, rounds to 2^24 in float32 but exact in float64

		floatKey := func(backing any) xpath3.AtomicValue {
			return xpath3.AtomicValue{TypeName: "Q{urn:test}myFloat", BaseType: xpath3.TypeFloat, Value: backing}
		}

		backings := []struct {
			name       string
			exactVal   any
			plusOneVal any
		}{
			{"float64", float64(exact), float64(plusOne)},
			{"float32", float32(exact), float32(plusOne)},
			{"FloatValue", xpath3.NewDouble(exact), xpath3.NewDouble(plusOne)},
		}

		for _, b := range backings {
			t.Run(b.name, func(t *testing.T) {
				// As xs:float, exact and plusOne share a key: a map storing the
				// exact key must resolve a lookup by plusOne.
				m := xpath3.NewMap([]xpath3.MapEntry{
					{Key: floatKey(b.exactVal), Value: xpath3.SingleString("found")},
					{Key: xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "sentinel"}, Value: xpath3.SingleString("s")},
				})
				require.True(t, m.Contains(floatKey(b.plusOneVal)),
					"xs:float 2^24+1 must hit the 2^24 key in single precision")
				v, ok := m.Get(floatKey(b.plusOneVal))
				require.True(t, ok)
				require.Equal(t, "found", v.Get(0).(xpath3.AtomicValue).StringVal())
			})
		}

		// As xs:double, the same two magnitudes remain distinct (no single-precision
		// rounding), guarding against over-collapsing the double path.
		dblKey := func(backing any) xpath3.AtomicValue {
			return xpath3.AtomicValue{TypeName: "Q{urn:test}myDouble", BaseType: xpath3.TypeDouble, Value: backing}
		}
		md := xpath3.NewMap([]xpath3.MapEntry{
			{Key: dblKey(float64(exact)), Value: xpath3.SingleString("exact")},
			{Key: xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "sentinel"}, Value: xpath3.SingleString("s")},
		})
		require.False(t, md.Contains(dblKey(float64(plusOne))),
			"xs:double 2^24 and 2^24+1 must remain distinct keys")
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
		require.Equal(t, int64(2), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("use last", func(t *testing.T) {
		merged, err := xpath3.MergeMaps([]xpath3.MapItem{m1, m2}, xpath3.MergeUseLast)
		require.NoError(t, err)
		v, _ := merged.Get(strKey("b"))
		require.Equal(t, int64(20), v.Get(0).(xpath3.AtomicValue).IntegerVal())
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
		require.Equal(t, int64(10), v.Get(0).(xpath3.AtomicValue).IntegerVal())

		v, err = a.Get(3)
		require.NoError(t, err)
		require.Equal(t, int64(30), v.Get(0).(xpath3.AtomicValue).IntegerVal())
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
		require.Equal(t, int64(99), v.Get(0).(xpath3.AtomicValue).IntegerVal())
		// Original unchanged
		v, _ = a.Get(2)
		require.Equal(t, int64(2), v.Get(0).(xpath3.AtomicValue).IntegerVal())
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
		require.Equal(t, int64(20), v.Get(0).(xpath3.AtomicValue).IntegerVal())
		v, _ = sub.Get(2)
		require.Equal(t, int64(30), v.Get(0).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("flatten", func(t *testing.T) {
		inner := xpath3.NewArray([]xpath3.Sequence{
			xpath3.SingleInteger(3),
			xpath3.SingleInteger(4),
		})
		a := xpath3.NewArray([]xpath3.Sequence{
			xpath3.SingleInteger(1),
			xpath3.SingleInteger(2),
			xpath3.ItemSlice{inner},
		})
		flat := a.Flatten()
		require.Equal(t, 4, flat.Len())
		require.Equal(t, int64(1), flat.Get(0).(xpath3.AtomicValue).IntegerVal())
		require.Equal(t, int64(4), flat.Get(3).(xpath3.AtomicValue).IntegerVal())
	})

	t.Run("constructor clones member sequences", func(t *testing.T) {
		member := xpath3.SingleString("original")
		a := xpath3.NewArray([]xpath3.Sequence{member})

		member.(xpath3.ItemSlice)[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testMutated}

		got, err := a.Get(1)
		require.NoError(t, err)
		require.Equal(t, "original", got.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("get returns cloned member sequence", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{xpath3.SingleString("original")})

		got, err := a.Get(1)
		require.NoError(t, err)
		got.(xpath3.ItemSlice)[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testMutated}

		again, err := a.Get(1)
		require.NoError(t, err)
		require.Equal(t, "original", again.Get(0).(xpath3.AtomicValue).StringVal())
	})

	t.Run("members returns cloned sequences", func(t *testing.T) {
		a := xpath3.NewArray([]xpath3.Sequence{xpath3.SingleString("original")})

		members := a.Members()
		members[0].(xpath3.ItemSlice)[0] = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testMutated}

		again, err := a.Get(1)
		require.NoError(t, err)
		require.Equal(t, "original", again.Get(0).(xpath3.AtomicValue).StringVal())
	})
}

func TestSequenceHelpers(t *testing.T) {
	t.Run("empty sequence", func(t *testing.T) {
		seq := xpath3.EmptySequence()
		require.Nil(t, seq)
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
		{lexicon.ValueTrue, xpath3.SingleBoolean(true), true, false},
		{lexicon.ValueFalse, xpath3.SingleBoolean(false), false, false},
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
	seq := xpath3.ItemSlice{
		xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: testHello},
		xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(42)},
	}
	atoms, err := xpath3.AtomizeSequence(seq)
	require.NoError(t, err)
	require.Len(t, atoms, 2)
	require.Equal(t, testHello, atoms[0].StringVal())
	require.Equal(t, int64(42), atoms[1].IntegerVal())
}

func TestAtomizeFunction(t *testing.T) {
	seq := xpath3.ItemSlice{xpath3.FunctionItem{Arity: 0, Name: testValue}}
	_, err := xpath3.AtomizeSequence(seq)
	require.Error(t, err)
}

// string() over the full XSD atomic type space drives atomicToString's
// per-type branches (gYear/gMonth/.., duration variants, base64/hex binary,
// QName, integer subtypes, date/time).
func TestAtomicToString_AllTypes(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		// gregorian types.
		{`string(xs:gYear("2020"))`, "2020"},
		{`string(xs:gMonth("--06"))`, "--06"},
		{`string(xs:gDay("---22"))`, "---22"},
		{`string(xs:gYearMonth("2020-06"))`, "2020-06"},
		{`string(xs:gMonthDay("--06-22"))`, "--06-22"},
		// date / time / dateTime.
		{`string(xs:date("2020-06-22"))`, "2020-06-22"},
		{`string(xs:time("12:34:56"))`, "12:34:56"},
		{`string(xs:dateTime("2020-06-22T12:34:56"))`, "2020-06-22T12:34:56"},
		// durations.
		{`string(xs:duration("P1Y2M3DT4H5M6S"))`, "P1Y2M3DT4H5M6S"},
		{`string(xs:dayTimeDuration("P1DT2H"))`, "P1DT2H"},
		{`string(xs:yearMonthDuration("P1Y2M"))`, "P1Y2M"},
		// binary.
		{`string(xs:base64Binary("aGk="))`, "aGk="},
		{`string(xs:hexBinary("48656C6C6F"))`, "48656C6C6F"},
		// integer subtypes.
		{`string(xs:byte("12"))`, "12"},
		{`string(xs:unsignedShort("300"))`, "300"},
		{`string(xs:positiveInteger("5"))`, "5"},
		// decimal / double / float / boolean / string-ish.
		{`string(xs:decimal("1.250"))`, "1.25"},
		{`string(xs:double("2.5"))`, "2.5"},
		{`string(xs:float("3.5"))`, "3.5"},
		{`string(true())`, wantTrue},
		{`string(xs:anyURI("http://x"))`, "http://x"},
		{`string(xs:NCName("abc"))`, "abc"},
		{`string(xs:token("  a  b  "))`, "a b"},
	}
	for _, tc := range cases {
		r, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		require.Equal(t, tc.want, r.StringValue(), tc.expr)
	}

	// QName via fn:QName, then string().
	r, err := evaluate(t.Context(), nil, `string(fn:QName("http://x", "p:local"))`)
	require.NoError(t, err)
	require.Equal(t, "p:local", r.StringValue())
}

func TestAtomicEquals(t *testing.T) {
	require.True(t, xpath3.AtomicEquals(intAtomic(3), intAtomic(3)))
	require.False(t, xpath3.AtomicEquals(intAtomic(3), intAtomic(4)))
	require.True(t, xpath3.AtomicEquals(strAtomic("x"), strAtomic("x")))
	// Incomparable types return false rather than erroring.
	require.False(t, xpath3.AtomicEquals(strAtomic("x"), intAtomic(1)))
}

func TestAtomicValue_StringIsNaN(t *testing.T) {
	s := intAtomic(7).String()
	require.Contains(t, s, "7")

	nan := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.NaN())}
	require.True(t, nan.IsNaN())

	notNaN := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(1.5)}
	require.False(t, notNaN.IsNaN())

	// Non-float types are never NaN.
	require.False(t, intAtomic(1).IsNaN())
}

func TestBuiltinIsSubtypeOf(t *testing.T) {
	require.True(t, xpath3.BuiltinIsSubtypeOf(xpath3.TypeInteger, xpath3.TypeDecimal))
	require.True(t, xpath3.BuiltinIsSubtypeOf(xpath3.TypeInteger, xpath3.TypeInteger))
	require.False(t, xpath3.BuiltinIsSubtypeOf(xpath3.TypeString, xpath3.TypeInteger))
}

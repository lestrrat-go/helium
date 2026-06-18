package xpath3

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMapNumericKeyOpSameKey pins the op:same-key semantics for numeric map keys.
// Per XPath 3.1 §17.1.1, two numeric values are the same key iff their EXACT decimal
// (value-space) expansions are mathematically equal — explicitly NOT the eq operator
// (which promotes to xs:double and is non-transitive). normalizeMapKey realizes this
// by bucketing on each value's exact rational string.
//
// Consequences pinned here:
//   - xs:float("0.1") and xs:decimal("0.1") are DIFFERENT keys: the float's exact
//     value is the binary-float32 rational, not 1/10. (eq would say true; same-key
//     says false.)
//   - xs:float("0.5") and xs:decimal("0.5") ARE the same key: both exactly 1/2.
//   - xs:float, xs:double and xs:decimal of 1/3 are three distinct keys (their exact
//     rationals all differ), matching QT3 op-same-key/same-key-008.
func TestMapNumericKeyOpSameKey(t *testing.T) {
	key := func(t *testing.T, s, typ string) AtomicValue {
		t.Helper()
		v, err := CastFromString(s, typ)
		require.NoError(t, err)
		return v
	}

	t.Run("float(0.1) and decimal(0.1) are distinct keys", func(t *testing.T) {
		floatKey := key(t, "0.1", TypeFloat)
		decKey := key(t, "0.1", TypeDecimal)
		// eq promotes to double and reports equal, but same-key must not.
		require.NotEqual(t, normalizeMapKey(floatKey), normalizeMapKey(decKey))
		m := NewMap([]MapEntry{
			{Key: floatKey, Value: SingleString("f")},
			{Key: decKey, Value: SingleString("d")},
		})
		require.Equal(t, 2, m.Size())
		got, ok := m.Get(decKey)
		require.True(t, ok)
		require.Equal(t, "d", got.Get(0).(AtomicValue).StringVal())
		got, ok = m.Get(floatKey)
		require.True(t, ok)
		require.Equal(t, "f", got.Get(0).(AtomicValue).StringVal())
	})

	t.Run("float(0.5) and decimal(0.5) are the same key", func(t *testing.T) {
		floatKey := key(t, "0.5", TypeFloat)
		decKey := key(t, "0.5", TypeDecimal)
		// Both have exact value 1/2, so they share a bucket.
		require.Equal(t, normalizeMapKey(floatKey), normalizeMapKey(decKey))
		m := NewMap([]MapEntry{
			{Key: floatKey, Value: SingleString("f")},
			{Key: decKey, Value: SingleString("d")},
		})
		got, ok := m.Get(floatKey)
		require.True(t, ok)
		require.Equal(t, "d", got.Get(0).(AtomicValue).StringVal(), "last entry wins on shared key")
	})

	t.Run("float, double, decimal of 1/3 are three distinct keys", func(t *testing.T) {
		fk := normalizeMapKey(AtomicValue{TypeName: TypeFloat, Value: NewFloat(1.0 / 3)})
		dk := normalizeMapKey(AtomicValue{TypeName: TypeDouble, Value: NewDouble(1.0 / 3)})
		ck := normalizeMapKey(AtomicValue{TypeName: TypeDecimal, Value: new(big.Rat).SetFrac64(1, 3)})
		require.NotEqual(t, fk, dk)
		require.NotEqual(t, fk, ck)
		require.NotEqual(t, dk, ck)
	})

	// Regression: xs:double(2^63) is an exact integer that exceeds int64 range.
	// math.MaxInt64 (2^63-1) rounds UP to 2^63 as a float64, so a naive
	// "f <= math.MaxInt64" guard let 2^63 through and int64(f) overflow-wrapped
	// to -2^63, colliding with the xs:integer(-2^63) key. The exact-rational
	// path must keep them distinct, while folding double(2^63) and decimal(2^63)
	// (same value-space value) into one key.
	t.Run("double 2^63 does not collide with integer -2^63", func(t *testing.T) {
		const twoTo63 = "9223372036854775808"     // 2^63, out of int64 range (max is 2^63-1)
		const negTwoTo63 = "-9223372036854775808" // -2^63, the int64 minimum
		dblKey := key(t, twoTo63, TypeDouble)     // xs:double(2^63)
		decKey := key(t, twoTo63, TypeDecimal)    // xs:decimal(2^63)
		intKey := key(t, negTwoTo63, TypeInteger) // xs:integer(-2^63)

		dblNK := normalizeMapKey(dblKey)
		decNK := normalizeMapKey(decKey)
		intNK := normalizeMapKey(intKey)

		// double(2^63) and decimal(2^63) are the same value-space value: same key.
		require.Equal(t, decNK, dblNK, "double(2^63) and decimal(2^63) must share a key")
		// double(2^63) is a positive value; it must NOT collide with integer(-2^63).
		require.NotEqual(t, intNK, dblNK, "double(2^63) must not collide with integer(-2^63)")

		m := NewMap([]MapEntry{
			{Key: dblKey, Value: SingleString("dbl")},
			{Key: intKey, Value: SingleString("int")},
		})
		require.Equal(t, 2, m.Size(), "2^63 double and -2^63 integer are distinct keys")
		got, ok := m.Get(dblKey)
		require.True(t, ok)
		require.Equal(t, "dbl", got.Get(0).(AtomicValue).StringVal())
		got, ok = m.Get(intKey)
		require.True(t, ok)
		require.Equal(t, "int", got.Get(0).(AtomicValue).StringVal())
		// decimal(2^63) keys into the double(2^63) slot, not the integer slot.
		got, ok = m.Get(decKey)
		require.True(t, ok)
		require.Equal(t, "dbl", got.Get(0).(AtomicValue).StringVal())
	})
}

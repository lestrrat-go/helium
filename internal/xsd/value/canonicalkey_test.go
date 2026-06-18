package value_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/stretchr/testify/require"
)

// TestCanonicalKey covers value-space canonicalization used for identity-
// constraint keys, focusing on whiteSpace handling per type and float vs double
// precision.
func TestCanonicalKey(t *testing.T) {
	key := func(s, typ string) string {
		k, _ := value.CanonicalKey(s, typ)
		return k
	}

	// xs:string is whiteSpace=preserve: leading/trailing space is significant,
	// so distinct strings must NOT collide.
	require.NotEqual(t, key("a", "string"), key("a ", "string"), `"a" and "a " must differ for xs:string`)
	require.Equal(t, "a ", key("a ", "string"), "xs:string must be preserved verbatim")

	// Collapse types (token, etc.) and list types collapse internal whitespace,
	// so separator-only differences ARE value-equal.
	require.Equal(t, key("a b", "token"), key("a  b", "token"), "xs:token collapses internal whitespace")
	require.Equal(t, key("x y", "IDREFS"), key("x  y", "IDREFS"), "xs:IDREFS collapses internal whitespace")
	require.Equal(t, key("x y", "IDREFS"), key(" x y ", "IDREFS"), "xs:IDREFS trims leading/trailing")

	// xs:float uses 32-bit IEEE: 16777216 and 16777217 round to the same float32
	// (2^24 boundary) and must collide, while as xs:double they stay distinct.
	require.Equal(t, key("16777216", "float"), key("16777217", "float"), "values equal in float32 must collide for xs:float")
	require.NotEqual(t, key("16777216", "double"), key("16777217", "double"), "distinct doubles must not collide for xs:double")

	// Signed zero: -0 and 0 are value-equal, so they must produce the same key.
	require.Equal(t, key("0", "double"), key("-0", "double"), "-0 and 0 must collide for xs:double")
	require.Equal(t, key("0", "float"), key("-0", "float"), "-0 and 0 must collide for xs:float")
	require.Equal(t, key("0.0", "double"), key("-0.0", "double"), "-0.0 and 0.0 must collide")

	// Huge expanded years use arbitrary-precision year keys, so timezone-
	// equivalent forms canonicalize to the same key (used for enumeration and
	// fixed-value identity), while distinct huge years must not collide.
	require.Equal(t,
		key(hugeYearPlus1+"Z", "gYear"),
		key(hugeYearPlus1+"+00:00", "gYear"),
		"TZ-equivalent huge gYear values must canonicalize equal")
	require.NotEqual(t,
		key(hugeYear, "gYear"),
		key(hugeYearPlus1, "gYear"),
		"distinct huge gYear values must not collide")
}

// TestCanonicalKeySignedYearInvalid verifies that a leading '+' on the year is
// not accepted as a valid date/dateTime lexical form: it must NOT canonicalize
// as valid, and must NOT produce a key equal to the unsigned form.
func TestCanonicalKeySignedYearInvalid(t *testing.T) {
	_ = t.Context()

	plusDate, okPlusDate := value.CanonicalKey("+2023-01-01", "date")
	require.False(t, okPlusDate, `"+2023-01-01" must not canonicalize as a valid xs:date`)

	unsignedDate, okUnsignedDate := value.CanonicalKey("2023-01-01", "date")
	require.True(t, okUnsignedDate, `"2023-01-01" must canonicalize as a valid xs:date`)
	require.NotEqual(t, unsignedDate, plusDate, `"+2023-01-01" must not produce the same key as "2023-01-01"`)

	plusDT, okPlusDT := value.CanonicalKey("+2023-01-01T00:00:00", "dateTime")
	require.False(t, okPlusDT, `"+2023-01-01T00:00:00" must not canonicalize as a valid xs:dateTime`)

	unsignedDT, okUnsignedDT := value.CanonicalKey("2023-01-01T00:00:00", "dateTime")
	require.True(t, okUnsignedDT, `"2023-01-01T00:00:00" must canonicalize as a valid xs:dateTime`)
	require.NotEqual(t, unsignedDT, plusDT, `"+2023-01-01T00:00:00" must not produce the same key as "2023-01-01T00:00:00"`)
}

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
}

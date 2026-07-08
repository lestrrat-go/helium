package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// XML 1.0 §3.3: "When more than one definition is provided for the same
// attribute of a given element type, the first declaration is binding and later
// declarations are ignored." A repeated <!ATTLIST> for the same attribute is a
// validity warning, not a fatal error — libxml2 accepts such documents. helium
// must accept them too and keep the first declaration's default.
func TestDuplicateAttlistDeclaration(t *testing.T) {
	t.Parallel()

	const doc = `<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ATTLIST doc a1 CDATA "first">
<!ATTLIST doc a1 CDATA "second">
]>
<doc></doc>`

	parsed, err := helium.NewParser().
		DefaultDTDAttributes(true).
		Parse(t.Context(), []byte(doc))
	require.NoError(t, err, "a duplicate ATTLIST attribute definition must not be a fatal error")
	require.NotNil(t, parsed)

	root := parsed.DocumentElement()
	require.NotNil(t, root)
	// The first declaration is binding: the defaulted value is "first".
	a1, ok := root.GetAttribute("a1")
	require.True(t, ok, "the defaulted attribute from the first ATTLIST must be present")
	require.Equal(t, "first", a1)
}

// A later, ignored duplicate declaration must not have its (possibly invalid)
// default value validated — §3.3 ignores the whole declaration, so an invalid
// default in the duplicate cannot abort the parse.
func TestDuplicateAttlistIgnoresInvalidDuplicateDefault(t *testing.T) {
	t.Parallel()

	const doc = `<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ATTLIST doc a1 CDATA "ok">
<!ATTLIST doc a1 NMTOKEN "not a token">
]>
<doc></doc>`

	parsed, err := helium.NewParser().
		DefaultDTDAttributes(true).
		Parse(t.Context(), []byte(doc))
	require.NoError(t, err, "the ignored duplicate's invalid default must not abort the parse")
	require.NotNil(t, parsed)
	a1, ok := parsed.DocumentElement().GetAttribute("a1")
	require.True(t, ok)
	require.Equal(t, "ok", a1, "the first (CDATA) declaration is binding")
}

// The first declaration's type governs attribute-value normalization; a later
// duplicate with a different type must not change it. First CDATA keeps explicit
// whitespace; a duplicate NMTOKEN must not cause collapsing.
func TestDuplicateAttlistTypeDoesNotAffectNormalization(t *testing.T) {
	t.Parallel()

	const doc = `<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ATTLIST doc a2 CDATA #IMPLIED>
<!ATTLIST doc a2 NMTOKEN #IMPLIED>
]>
<doc a2="  x   y  "></doc>`

	parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
	require.NoError(t, err)
	a2, ok := parsed.DocumentElement().GetAttribute("a2")
	require.True(t, ok)
	// CDATA (the first, binding type) preserves the explicit whitespace; had the
	// duplicate NMTOKEN won, the value would have been collapsed to "x y".
	require.Equal(t, "  x   y  ", a2)
}

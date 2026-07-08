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

package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// XML 1.0 §3.3: AttlistDecl ::= '<!ATTLIST' S Name AttDef* S? '>'. AttDef* is
// zero-or-more, so an empty attribute-list declaration `<!ATTLIST name>` (with
// or without trailing whitespace) declares no attributes and is well-formed.
func TestEmptyAttlistDeclaration(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		doc  string
	}{
		{
			name: "no trailing space",
			doc: `<!DOCTYPE r [
<!ELEMENT r EMPTY>
<!ATTLIST r>
]>
<r/>`,
		},
		{
			name: "trailing space",
			doc: `<!DOCTYPE r [
<!ELEMENT r EMPTY>
<!ATTLIST r >
]>
<r/>`,
		},
		{
			name: "empty then non-empty for same element",
			doc: `<!DOCTYPE r [
<!ELEMENT r (#PCDATA)>
<!ATTLIST r>
<!ATTLIST r a CDATA #IMPLIED>
]>
<r a="x">y</r>`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := helium.NewParser().Parse(t.Context(), []byte(tc.doc))
			require.NoError(t, err, "an empty <!ATTLIST> is well-formed and must parse")
			require.NotNil(t, parsed)
			require.NotNil(t, parsed.DocumentElement())
		})
	}
}

// A non-empty ATTLIST still parses and its declared default is applied, proving
// the empty-list handling did not regress the AttDef loop.
func TestNonEmptyAttlistStillWorks(t *testing.T) {
	t.Parallel()

	const doc = `<!DOCTYPE r [
<!ELEMENT r (#PCDATA)>
<!ATTLIST r a CDATA "def">
]>
<r>x</r>`

	parsed, err := helium.NewParser().
		DefaultDTDAttributes(true).
		Parse(t.Context(), []byte(doc))
	require.NoError(t, err)
	a, ok := parsed.DocumentElement().GetAttribute("a")
	require.True(t, ok, "the declared default attribute must be present")
	require.Equal(t, "def", a)
}

// An ATTLIST with no element Name at all is malformed and must still be
// rejected — the mandatory `S Name` after `<!ATTLIST` is unaffected by allowing
// an empty AttDef list.
func TestAttlistWithoutElementNameRejected(t *testing.T) {
	t.Parallel()

	const doc = `<!DOCTYPE r [
<!ELEMENT r EMPTY>
<!ATTLIST>
]>
<r/>`

	_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
	require.Error(t, err, "an <!ATTLIST> with no element name must be rejected")
}

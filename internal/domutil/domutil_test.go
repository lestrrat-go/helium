package domutil_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/stretchr/testify/require"
)

// TestFindElementsByID pins the FROZEN ID-name rule FindElementsByID and
// BuildIDIndex share: a DTD/schema-declared ID-typed attribute, xml:id, and
// the "id" attribute token in the casings Id/ID/id are recognized, while a
// distinct convention token such as wsu:Id is deliberately NOT recognized by
// name. This rule is a security boundary against XML Signature Wrapping, so
// both functions are exercised against the identical set of documents.
func TestFindElementsByID(t *testing.T) {
	t.Parallel()

	const wantID = "foo"
	testcases := []struct {
		name  string
		xml   string
		id    string
		count int // number of matching elements expected
	}{
		{
			name:  "capitalized Id",
			xml:   `<root><target Id="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "uppercase ID",
			xml:   `<root><target ID="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "lowercase id",
			xml:   `<root><target id="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "xml:id",
			xml:   `<root><target xml:id="foo"/></root>`,
			id:    wantID,
			count: 1,
		},
		{
			name:  "no match",
			xml:   `<root><target id="bar"/></root>`,
			id:    wantID,
			count: 0,
		},
		{
			name:  "distinct token not recognized",
			xml:   `<root xmlns:wsu="http://x"><target wsu:Id="foo"/></root>`,
			id:    wantID,
			count: 0,
		},
		{
			name:  "duplicate id is ambiguous",
			xml:   `<root><a Id="foo"/><b id="foo"/></root>`,
			id:    wantID,
			count: 2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.xml))
			require.NoError(t, err)

			matches := domutil.FindElementsByID(doc.DocumentElement(), tc.id)
			require.Len(t, matches, tc.count)

			index := domutil.BuildIDIndex(doc.DocumentElement())
			require.Len(t, index[tc.id], tc.count)
		})
	}
}

// TestFindElementsByIDDTDDeclared proves the rule also recognizes an
// attribute typed ID by a DTD ATTLIST declaration, independent of its name —
// here "eid", which none of the name-based casings match.
func TestFindElementsByIDDTDDeclared(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (item*)>
  <!ELEMENT item (#PCDATA)>
  <!ATTLIST item eid ID #IMPLIED>
]>
<root>
  <item eid="x1">alpha</item>
  <item eid="x2">beta</item>
</root>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	matches := domutil.FindElementsByID(doc.DocumentElement(), "x1")
	require.Len(t, matches, 1)
	require.Equal(t, "item", matches[0].LocalName())

	index := domutil.BuildIDIndex(doc.DocumentElement())
	require.Len(t, index["x1"], 1)
	require.Equal(t, "item", index["x1"][0].LocalName())
	require.Len(t, index["x2"], 1)
}

// TestFindElementsByIDNilRoot proves a nil root is a no-match, not a panic.
func TestFindElementsByIDNilRoot(t *testing.T) {
	t.Parallel()

	require.Empty(t, domutil.FindElementsByID(nil, "foo"))
	require.Empty(t, domutil.BuildIDIndex(nil))
}

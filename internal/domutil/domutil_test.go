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

// TestBuildIDIndexMultipleIDAttributes proves BuildIDIndex records every id on
// an element carrying more than one recognized ID attribute, rather than only
// the first. FindElementsByID matches such an element for each of its ids, so
// an index that kept only the first would disagree with it.
func TestBuildIDIndexMultipleIDAttributes(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><x Id="a" id="b"/></root>`))
	require.NoError(t, err)

	root := doc.DocumentElement()
	index := domutil.BuildIDIndex(root)

	for _, id := range []string{"a", "b"} {
		require.Len(t, index[id], 1, "id %q", id)
		require.Equal(t, "x", index[id][0].LocalName(), "id %q", id)
		require.Equal(t, domutil.FindElementsByID(root, id), index[id], "id %q", id)
	}
}

// TestBuildIDIndexShadowedDuplicate proves a leading ID attribute on one
// element cannot hide a duplicate id shared with another element. Both
// elements carry id "sig", so the len(index[id]) > 1 check the doc comment
// tells callers to use must still see the ambiguity — the same ambiguity
// FindElementsByID reports. Masking it would be an XML Signature Wrapping
// hole.
func TestBuildIDIndexShadowedDuplicate(t *testing.T) {
	t.Parallel()

	const input = `<root><evil Id="decoy" id="sig"/><good id="sig"/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	root := doc.DocumentElement()
	index := domutil.BuildIDIndex(root)

	require.Len(t, index["sig"], 2)
	require.Equal(t, "evil", index["sig"][0].LocalName())
	require.Equal(t, "good", index["sig"][1].LocalName())
	require.Equal(t, domutil.FindElementsByID(root, "sig"), index["sig"])
	require.Equal(t, domutil.FindElementsByID(root, "decoy"), index["decoy"])
}

// TestIDIndexAgreesWithFindElementsByID is the parity check the two functions
// exist to satisfy: for any element shape, BuildIDIndex(root)[id] must be the
// same elements in the same order as FindElementsByID(root, id). The candidate
// ids are listed per document rather than read back from the index, so an
// index that dropped an id entirely cannot hide behind its own key set.
// xmldsig1 and xmlenc1 must never disagree about which elements carry an id;
// that disagreement is an XML Signature Wrapping hole.
func TestIDIndexAgreesWithFindElementsByID(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		xml  string
		ids  []string
	}{
		{
			name: "two ID attributes on one element",
			xml:  `<root><x Id="a" id="b"/></root>`,
			ids:  []string{"a", "b", "root", "missing"},
		},
		{
			name: "same id twice on one element",
			xml:  `<root><x Id="dup" id="dup"/></root>`,
			ids:  []string{"dup", "missing"},
		},
		{
			name: "three casings plus xml:id on one element",
			xml:  `<root><x Id="a" ID="b" id="c" xml:id="d"/></root>`,
			ids:  []string{"a", "b", "c", "d"},
		},
		{
			name: "recognized and unrecognized attributes mixed",
			xml:  `<root xmlns:wsu="http://x"><x wsu:Id="skipped" name="n" id="kept"/></root>`,
			ids:  []string{"kept", "skipped", "n"},
		},
		{
			name: "shadowing first ID attribute over a duplicate",
			xml:  `<root><evil Id="decoy" id="sig"/><good id="sig"/></root>`,
			ids:  []string{"sig", "decoy"},
		},
		{
			name: "duplicate across nesting levels",
			xml:  `<root id="dup"><a Id="dup"><b xml:id="dup"/></a></root>`,
			ids:  []string{"dup"},
		},
		{
			name: "whitespace-padded values collapse to one id",
			xml:  `<root><x Id=" pad " id="pad"/><y id="  pad"/></root>`,
			ids:  []string{"pad", " pad ", ""},
		},
		{
			name: "no attributes anywhere",
			xml:  `<root><a><b/></a></root>`,
			ids:  []string{"", "a"},
		},
		{
			name: "empty id value",
			xml:  `<root><x id="" Id="e"/></root>`,
			ids:  []string{"", "e"},
		},
		{
			name: "DTD-declared ID alongside a name-based one",
			xml: `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (item*)>
  <!ELEMENT item (#PCDATA)>
  <!ATTLIST item eid ID #IMPLIED>
]>
<root><item eid="x1" id="x2">alpha</item><item eid="x2">beta</item></root>`,
			ids: []string{"x1", "x2"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.xml))
			require.NoError(t, err)

			root := doc.DocumentElement()
			index := domutil.BuildIDIndex(root)

			for _, id := range tc.ids {
				require.Equal(t, domutil.FindElementsByID(root, id), index[id], "id %q", id)
			}
		})
	}
}

// TestFindElementsByIDNilRoot proves a nil root is a no-match, not a panic.
func TestFindElementsByIDNilRoot(t *testing.T) {
	t.Parallel()

	require.Empty(t, domutil.FindElementsByID(nil, "foo"))
	require.Empty(t, domutil.BuildIDIndex(nil))
}

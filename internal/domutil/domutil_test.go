package domutil_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/stretchr/testify/require"
)

var namespaceLookupSink string

func TestLookupNSPrefixURI(t *testing.T) {
	t.Parallel()

	t.Run("nearest declarations shadow ancestors", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(
			`<root xmlns="urn:outer" xmlns:p="urn:p-outer"><child xmlns="urn:inner" xmlns:p="urn:p-inner"><leaf/></child></root>`,
		))
		require.NoError(t, err)

		child := doc.DocumentElement().FirstChild().(*helium.Element) //nolint:forcetypeassert
		leaf := child.FirstChild().(*helium.Element)                  //nolint:forcetypeassert

		uri, found := domutil.LookupNSPrefixURI(leaf, "")
		require.True(t, found)
		require.Equal(t, "urn:inner", uri)

		uri, found = domutil.LookupNSPrefixURI(leaf, "p")
		require.True(t, found)
		require.Equal(t, "urn:p-inner", uri)
	})

	t.Run("undeclarations stop the ancestor walk", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(
			`<?xml version="1.1"?><root xmlns="urn:outer" xmlns:p="urn:p"><child xmlns="" xmlns:p=""><leaf/></child></root>`,
		))
		require.NoError(t, err)

		child := doc.DocumentElement().FirstChild().(*helium.Element) //nolint:forcetypeassert
		leaf := child.FirstChild().(*helium.Element)                  //nolint:forcetypeassert

		for _, prefix := range []string{"", "p"} {
			uri, found := domutil.LookupNSPrefixURI(leaf, prefix)
			require.True(t, found, "prefix %q", prefix)
			require.Empty(t, uri, "prefix %q", prefix)
		}
	})

	t.Run("xml is not implicitly declared", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)

		_, found := domutil.LookupNSPrefixURI(root, "xml")
		require.False(t, found)

		require.NoError(t, root.DeclareNamespace("xml", "http://www.w3.org/XML/1998/namespace"))
		uri, found := domutil.LookupNSPrefixURI(root, "xml")
		require.True(t, found)
		require.Equal(t, "http://www.w3.org/XML/1998/namespace", uri)
	})

	t.Run("CleanNamespaces controls retained declarations", func(t *testing.T) {
		const input = `<root xmlns:p="urn:p"><child xmlns:p="urn:p"><leaf/></child></root>`
		for _, clean := range []bool{false, true} {
			doc, err := helium.NewParser().CleanNamespaces(clean).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			root := doc.DocumentElement()
			child := root.FirstChild().(*helium.Element) //nolint:forcetypeassert
			leaf := child.FirstChild().(*helium.Element) //nolint:forcetypeassert
			require.True(t, root.RemoveNamespaceByPrefix("p"))

			uri, found := domutil.LookupNSPrefixURI(leaf, "p")
			if clean {
				require.False(t, found)
				continue
			}
			require.True(t, found)
			require.Equal(t, "urn:p", uri)
		}
	})

	t.Run("non-element start uses its element ancestor", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:p="urn:p">text</root>`))
		require.NoError(t, err)

		uri, found := domutil.LookupNSPrefixURI(doc.DocumentElement().FirstChild(), "p")
		require.True(t, found)
		require.Equal(t, "urn:p", uri)
	})
}

func TestLookupNSPrefixURIAllocationScaling(t *testing.T) {
	measure := func(width int) float64 {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		for i := range width {
			require.NoError(t, root.DeclareNamespace(fmt.Sprintf("p%d", i), fmt.Sprintf("urn:%d", i)))
		}
		children := make([]*helium.Element, 0, width)
		for i := range width {
			child, err := doc.CreateElement(fmt.Sprintf("child%d", i))
			require.NoError(t, err)
			require.NoError(t, root.AddChild(child))
			children = append(children, child)
		}
		prefix := fmt.Sprintf("p%d", width-1)

		return testing.AllocsPerRun(20, func() {
			for _, child := range children {
				uri, found := domutil.LookupNSPrefixURI(child, prefix)
				if !found {
					panic("namespace not found")
				}
				namespaceLookupSink = uri
			}
		})
	}

	small := measure(64)
	large := measure(512)
	t.Logf("namespace lookup allocations: width=64 %.0f, width=512 %.0f", small, large)
	require.LessOrEqual(t, large, small*2+1,
		"an eightfold wider namespace table and child list must keep lookup allocations constant")
}

func TestLookupNSURI(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<root xmlns:q="urn:shared"><child xmlns:p="urn:shared"><leaf/></child></root>`,
	))
	require.NoError(t, err)
	child := doc.DocumentElement().FirstChild().(*helium.Element) //nolint:forcetypeassert
	leaf := child.FirstChild().(*helium.Element)                  //nolint:forcetypeassert

	ns, found := domutil.LookupNSURI(leaf, "urn:shared")
	require.True(t, found)
	require.Equal(t, "p", ns.Prefix(), "the nearest declaration must win")

	_, found = domutil.LookupNSURI(leaf, "http://www.w3.org/XML/1998/namespace")
	require.False(t, found, "the bare declaration lookup must not synthesize xml")
}

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

			index, err := domutil.BuildIDIndex(t.Context(), doc.DocumentElement())
			require.NoError(t, err)
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

	index, err := domutil.BuildIDIndex(t.Context(), doc.DocumentElement())
	require.NoError(t, err)
	require.Len(t, index["x1"], 1)
	require.Equal(t, "item", index["x1"][0].LocalName())
	require.Len(t, index["x2"], 1)
}

// TestBuildIDIndexMultipleIDAttributes proves BuildIDIndex records every id on
// an element carrying more than one recognized ID attribute, down to the
// last. FindElementsByID matches such an element for each of its ids, so
// an index that kept only the first would disagree with it.
func TestBuildIDIndexMultipleIDAttributes(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><x Id="a" id="b"/></root>`))
	require.NoError(t, err)

	root := doc.DocumentElement()
	index, err := domutil.BuildIDIndex(t.Context(), root)
	require.NoError(t, err)

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
	index, err := domutil.BuildIDIndex(t.Context(), root)
	require.NoError(t, err)

	require.Len(t, index["sig"], 2)
	require.Equal(t, "evil", index["sig"][0].LocalName())
	require.Equal(t, "good", index["sig"][1].LocalName())
	require.Equal(t, domutil.FindElementsByID(root, "sig"), index["sig"])
	require.Equal(t, domutil.FindElementsByID(root, "decoy"), index["decoy"])
}

// TestBuildIDIndexCancelled proves the walk answers a cancelled caller instead
// of running to the end of a document it did not write, and that it hands back
// no index at all. A partial one would let a caller read an absent id out of
// it.
func TestBuildIDIndexCancelled(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a id="x"/><b id="y"/></root>`))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	index, err := domutil.BuildIDIndex(ctx, doc.DocumentElement())
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, index)
}

// TestIDIndexAgreesWithFindElementsByID is the parity check the two functions
// exist to satisfy: for any element shape, BuildIDIndex(ctx, root)[id] must be the
// same elements in the same order as FindElementsByID(root, id). The candidate
// ids are listed per document, never read back from the index, so an
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
			index, err := domutil.BuildIDIndex(t.Context(), root)
			require.NoError(t, err)

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
	index, err := domutil.BuildIDIndex(t.Context(), nil)
	require.NoError(t, err)
	require.Empty(t, index)
}

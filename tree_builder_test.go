package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestTreeBuilderSAXPath drives a content-rich document through the generic SAX
// callback path so the TreeBuilder's SAX2Handler methods build the tree.
func TestTreeBuilderSAXPath(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA|child)*>
<!ELEMENT child (#PCDATA)>
<!ATTLIST doc id ID #IMPLIED>
<!ENTITY greeting "hello">
<!ENTITY img SYSTEM "img.gif" NDATA gif>
<!NOTATION gif SYSTEM "viewer.exe">
]>
<doc id="d1">
  <?pi-target pi-data?>
  <!-- a comment -->
  <child>text &greeting; more</child>
  <![CDATA[ raw <data> & stuff ]]>
</doc>`

	handler := &saxTreeBuilder{TreeBuilder: helium.NewTreeBuilder()}
	doc, err := helium.NewParser().
		SubstituteEntities(true).
		SAXHandler(handler).
		Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse through SAX path succeeds")
	require.NotNil(t, doc)

	root := doc.DocumentElement()
	require.NotNil(t, root)
	require.Equal(t, "doc", root.Name())

	// The internal subset must have been built via the SAX DTD callbacks.
	dtd := doc.IntSubset()
	require.NotNil(t, dtd)
	_, ok := dtd.LookupEntity("greeting")
	require.True(t, ok, "greeting entity declared via SAX EntityDecl")
	_, ok = dtd.LookupNotation("gif")
	require.True(t, ok, "gif notation declared via SAX NotationDecl")

	// Serialize the SAX-built tree to confirm the structure is intact.
	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<child>")
	require.Contains(t, out, "<![CDATA[")
	require.Contains(t, out, "<?pi-target")
	require.Contains(t, out, "<!--")
}

// TestTreeBuilderSAXPathNoEntitySubstitution drives the SAX path without entity
// substitution so the Reference callback materializes entity-reference nodes.
func TestTreeBuilderSAXPathNoEntitySubstitution(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY greeting "hello">
]>
<doc>before &greeting; after</doc>`

	handler := &saxTreeBuilder{TreeBuilder: helium.NewTreeBuilder()}
	doc, err := helium.NewParser().
		SAXHandler(handler).
		Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	require.NotNil(t, doc)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "&greeting;")
}

// TestTreeBuilderSAXPathNamespaces drives namespaced markup through the SAX
// path so StartElementNS/EndElementNS bind and serialize namespaces correctly.
func TestTreeBuilderSAXPathNamespaces(t *testing.T) {
	t.Parallel()

	const src = `<root xmlns:a="urn:a" xmlns="urn:default">` +
		`<a:child attr="v">text</a:child>` +
		`<plain/>` +
		`</root>`

	handler := &saxTreeBuilder{TreeBuilder: helium.NewTreeBuilder()}
	doc, err := helium.NewParser().
		SAXHandler(handler).
		Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, `xmlns:a="urn:a"`)
	require.Contains(t, out, `xmlns="urn:default"`)
	require.Contains(t, out, "<a:child")
}

// saxTreeBuilder embeds *helium.TreeBuilder without being a *TreeBuilder
// itself. Because the parser only takes its fast (TreeBuilder-direct) path when
// the supplied SAX handler is *exactly* a *TreeBuilder, wrapping it like this
// forces the parser through the generic SAX2 callback path, exercising the
// TreeBuilder's SAX2Handler methods (Comment, CDataBlock, Characters,
// ProcessingInstruction, Reference, entity/DTD declaration callbacks, ...) that
// the fast path bypasses.
type saxTreeBuilder struct {
	*helium.TreeBuilder
}

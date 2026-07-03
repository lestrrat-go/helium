package helium_test

import (
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestCopyDTDInfo copies the internal-subset DTD declarations from one document
// into another via CopyDTDInfo.
func TestCopyDTDInfo(t *testing.T) {
	t.Parallel()

	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)
	require.NotNil(t, src.IntSubset())

	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	helium.CopyDTDInfo(src, dst)

	require.NotNil(t, dst.IntSubset(), "CopyDTDInfo populates the destination internal subset")
	_, ok := dst.IntSubset().LookupNotation("gif")
	require.True(t, ok, "notation copied via CopyDTDInfo")

	// nil arguments are a no-op (no panic).
	helium.CopyDTDInfo(nil, dst)
	helium.CopyDTDInfo(src, nil)
}

// TestCopyExtSubset copies an external DTD subset between documents.
func TestCopyExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := dir + "/ext.dtd"
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>
<!NOTATION gif SYSTEM "viewer.exe">
<!ENTITY ext SYSTEM "data.xml">`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`

	src, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	require.NotNil(t, src.ExtSubset())

	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	helium.CopyExtSubset(src, dst)
	require.NotNil(t, dst.ExtSubset(), "external subset copied")

	// nil arguments are a no-op.
	helium.CopyExtSubset(nil, dst)
	helium.CopyExtSubset(src, nil)
}

// TestCopyDocWithMixedChildren builds a document whose root element holds every
// leaf child type (Text, CDATA, Comment, PI, EntityRef), then deep-copies it via
// CopyDoc so the per-node-type branches of the deep copier's copyNode are all
// exercised.
func TestCopyDocWithMixedChildren(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.AddChild(doc.CreateText([]byte("text"))))
	require.NoError(t, root.AddChild(doc.CreateCDATASection([]byte("<cdata>"))))
	require.NoError(t, root.AddChild(doc.CreateComment([]byte("comment"))))
	require.NoError(t, root.AddChild(doc.CreatePI("target", "data")))
	ref, err := doc.CreateReference("amp")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(ref))

	// A top-level comment and PI exercise the document-level copyChildren too.
	require.NoError(t, doc.AddChild(doc.CreateComment([]byte("top-comment"))))
	require.NoError(t, doc.AddChild(doc.CreatePI("toppi", "x")))

	cp, err := helium.CopyDoc(doc)
	require.NoError(t, err)
	require.NotNil(t, cp)

	cpRoot := cp.DocumentElement()
	require.NotNil(t, cpRoot)

	// Walk the copied children and confirm each node type round-tripped.
	var kinds []helium.ElementType
	for c := cpRoot.FirstChild(); c != nil; c = c.NextSibling() {
		kinds = append(kinds, c.Type())
	}
	require.Contains(t, kinds, helium.TextNode)
	require.Contains(t, kinds, helium.CDATASectionNode)
	require.Contains(t, kinds, helium.CommentNode)
	require.Contains(t, kinds, helium.ProcessingInstructionNode)
	require.Contains(t, kinds, helium.EntityRefNode)
}

// TestCopyDocWithDTD parses a document that has a rich internal DTD subset and
// deep-copies it, then serializes both, exercising copy.go, copy_core.go and
// the DTD writer paths.
func TestCopyDocWithDTD(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)

	orig, err := helium.WriteString(doc)
	require.NoError(t, err)

	cp, err := helium.CopyDoc(doc)
	require.NoError(t, err)
	copied, err := helium.WriteString(cp)
	require.NoError(t, err)

	// A faithful deep copy round-trips identically.
	require.Equal(t, orig, copied)

	// CopyDoc(nil) is rejected.
	_, err = helium.CopyDoc(nil)
	require.Error(t, err)
}

// TestCopyDocEntityBearingAttributes exercises the deep-copy attribute path with
// values that the parser has already entity-resolved. Copying such values with
// the value-PARSING setters (SetAttribute/SetAttributeNS) would re-interpret a
// bare '&'/'<' — raising "entity was unterminated" for a value that came from
// '&amp;', and silently double-resolving '&amp;amp;'. The literal setters store
// the resolved value as-is and let the serializer re-escape it, so a CopyDoc
// round-trips byte-for-byte identically.
func TestCopyDocEntityBearingAttributes(t *testing.T) {
	t.Parallel()

	src := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<root amp="x&amp;y" lt="a&lt;b" gt="a&gt;b" q="say &quot;hi&quot;" num="&#65;BC" dbl="a&amp;amp;b" p:ns="u&amp;v" xmlns:p="urn:p"/>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	// Sanity: the parser resolved the entity references, so the in-memory value
	// carries a bare '&'/'<'/'>' — exactly the shape that breaks a re-parsing copy.
	root := doc.DocumentElement()
	require.NotNil(t, root)
	amp, ok := root.FindAttribute(helium.LocalNamePredicate("amp"))
	require.True(t, ok)
	require.Equal(t, "x&y", amp.Value(), "parser resolved &amp; to a bare &")

	orig, err := helium.WriteString(doc)
	require.NoError(t, err)

	cp, err := helium.CopyDoc(doc)
	require.NoError(t, err, "CopyDoc must not choke on entity-resolved attribute values")

	copied, err := helium.WriteString(cp)
	require.NoError(t, err)

	// The copy serializes byte-for-byte identically to the original: the
	// serializer re-escapes '&'/'<' on both, and the resolved values are stored
	// literally rather than re-parsed.
	require.Equal(t, orig, copied)

	// And CopyNode of just the element (the other deepCopier entry point) too.
	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	cpNode, err := helium.CopyNode(root, dst)
	require.NoError(t, err)
	cpElem, ok := helium.AsNode[*helium.Element](cpNode)
	require.True(t, ok)
	cpAmp, ok := cpElem.FindAttribute(helium.LocalNamePredicate("amp"))
	require.True(t, ok)
	require.Equal(t, "x&y", cpAmp.Value(), "copied value stays resolved, not re-parsed or double-escaped")
	cpDbl, ok := cpElem.FindAttribute(helium.LocalNamePredicate("dbl"))
	require.True(t, ok)
	require.Equal(t, "a&amp;b", cpDbl.Value(), "double-escaped source '&amp;amp;' resolves once to '&amp;', copied literally")
}

// TestCopyNodeVariants covers CopyNode across several node types.
func TestCopyNodeVariants(t *testing.T) {
	t.Parallel()
	src := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

	root := src.CreateElement("root")
	require.NoError(t, src.AddChild(root))

	text := src.CreateText([]byte("hi"))
	comment := src.CreateComment([]byte("c"))
	cdata := src.CreateCDATASection([]byte("data"))
	pi := src.CreatePI("target", "value")

	for _, n := range []helium.Node{text, comment, cdata, pi, root} {
		cp, err := helium.CopyNode(n, dst)
		require.NoError(t, err, "CopyNode(%s)", n.Type())
		require.Equal(t, n.Type(), cp.Type())
	}

	// Copy the whole document via CopyNode (delegates to CopyDoc).
	cp, err := helium.CopyNode(src, dst)
	require.NoError(t, err)
	require.Equal(t, helium.DocumentNode, cp.Type())
}

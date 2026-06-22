package helium_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestDocumentAccessors exercises the small Document getter/setter methods that
// are otherwise only touched indirectly.
func TestDocumentAccessors(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitYes)
	require.Equal(t, "UTF-8", doc.Encoding())
	require.Equal(t, "UTF-8", doc.RawEncoding())
	require.Equal(t, "1.0", doc.Version())

	doc.SetEncoding("ISO-8859-1")
	require.Equal(t, "ISO-8859-1", doc.Encoding())
	require.Equal(t, "ISO-8859-1", doc.RawEncoding())

	// Document with no encoding synthesizes "utf8" for Encoding but empty for raw.
	d2 := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	require.Equal(t, "utf8", d2.Encoding())
	require.Equal(t, "", d2.RawEncoding())

	doc.SetURL("http://example.com/doc.xml")
	require.Equal(t, "http://example.com/doc.xml", doc.URL())

	doc.SetProperties(helium.DocHTML)
	require.True(t, doc.HasProperty(helium.DocHTML))
	require.Equal(t, helium.DocHTML, doc.Properties())

	doc.SetSkipIDs(true)
	require.True(t, doc.SkipIDs())
	doc.SetSkipIDs(false)
	require.False(t, doc.SkipIDs())

	require.Equal(t, helium.DocumentStandaloneType(helium.StandaloneExplicitYes), doc.Standalone())

	// AddSibling/Replace on a document are rejected.
	require.Error(t, doc.AddSibling(doc.CreateElement("x")))
	require.Error(t, doc.Replace())

	// SetTreeDoc on a document is a no-op-ish but must not panic.
	doc.SetTreeDoc(doc)
}

func TestNewHTMLDocument(t *testing.T) {
	t.Parallel()
	doc := helium.NewHTMLDocument()
	require.Equal(t, helium.HTMLDocumentNode, doc.Type())
	require.True(t, doc.HasProperty(helium.DocHTML))
}

// TestDocumentFree builds a parsed document and then frees its slabs. This must
// be safe and idempotent.
func TestDocumentFree(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)
	doc.Free()
	doc.Free() // idempotent
}

// TestInternalSubsetAccessors covers DTD construction and its accessor methods.
func TestInternalSubsetAccessors(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "-//Example//DTD//EN", "example.dtd")
	require.NoError(t, err)

	require.Equal(t, "-//Example//DTD//EN", dtd.ExternalID())
	require.Equal(t, "example.dtd", dtd.SystemID())
	require.Same(t, dtd, doc.IntSubset())

	// Exercise the DTD node-interface methods (delegating wrappers). They must
	// not panic; their success/error is implementation-defined for an
	// already-attached internal subset.
	_ = dtd.AddSibling(doc.CreateElement("x"))
	_ = dtd.Replace()
	dtd.SetTreeDoc(doc)
	dtd.Free()
}

// TestDTDEntityAndNotation exercises AddEntity/AddNotation/ForEachEntity and the
// resulting Entity/Notation accessor methods.
func TestDTDEntityAndNotation(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	gen, err := dtd.AddEntity("greeting", enum.InternalGeneralEntity, "", "", "Hello")
	require.NoError(t, err)
	require.Equal(t, enum.InternalGeneralEntity, gen.EntityType())
	require.Equal(t, []byte("Hello"), gen.Content())

	// External unparsed entity exposes systemID/externalID/URI.
	img, err := dtd.AddEntity("img", enum.ExternalGeneralUnparsedEntity, "", "img.gif", "gif")
	require.NoError(t, err)
	require.Equal(t, "img.gif", img.SystemID())
	require.Equal(t, "", img.ExternalID())
	require.Equal(t, "img.gif", img.URI()) // falls back to systemID

	// First-definition-wins: redeclaring returns the existing entity.
	again, err := dtd.AddEntity("greeting", enum.InternalGeneralEntity, "", "", "Goodbye")
	require.NoError(t, err)
	require.Same(t, gen, again)

	// Redeclaring a predefined entity with the wrong content is rejected.
	_, err = dtd.AddEntity("lt", enum.InternalGeneralEntity, "", "", "wrong")
	require.Error(t, err)

	// Predefined entity type cannot be registered.
	_, err = dtd.AddEntity("x", enum.InternalPredefinedEntity, "", "", "y")
	require.Error(t, err)

	// ForEachEntity visits the general entities.
	seen := map[string]bool{}
	dtd.ForEachEntity(func(name string, ent *helium.Entity) {
		seen[name] = true
	})
	require.True(t, seen["greeting"])
	require.True(t, seen["img"])

	nota, err := dtd.AddNotation("gif", "", "viewer.exe")
	require.NoError(t, err)
	require.Equal(t, helium.NotationNode, nota.Type())

	// Redefinition of a notation is rejected.
	_, err = dtd.AddNotation("gif", "", "other.exe")
	require.Error(t, err)

	// Document.AddEntity delegates to the internal subset.
	_, err = doc.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
	require.NoError(t, err)

	// Document.AddEntity on a doc without an internal subset errors.
	bare := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	_, err = bare.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
	require.Error(t, err)
}

// TestEntityNodeMethods covers the Entity node-interface methods.
func TestEntityNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)
	ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	ent.SetOrig("&e;")
	require.False(t, ent.Checked())
	ent.MarkChecked()
	require.True(t, ent.Checked())

	ent.SetTreeDoc(doc)
	require.NoError(t, ent.AppendText([]byte("more")))
}

// TestNotationNodeMethods covers Notation node-interface methods.
func TestNotationNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)
	nota, err := dtd.AddNotation("n", "pub", "sys")
	require.NoError(t, err)

	nota.SetTreeDoc(doc)
	nota.Free()
	require.NoError(t, nota.AppendText([]byte("x")))
}

// TestXIncludeMarker exercises the XIncludeMarker node type and its methods.
func TestXIncludeMarker(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	m := helium.NewXIncludeMarker(doc, helium.XIncludeStartNode, "include")
	require.Equal(t, helium.XIncludeStartNode, m.Type())
	require.Equal(t, "include", m.Name())

	child := doc.CreateText([]byte("hello"))
	require.NoError(t, m.AddChild(child))
	require.NoError(t, m.AppendText([]byte(" world")))
	m.SetTreeDoc(doc)
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

// TestWriterOptions exercises the Writer option toggles and serialization paths.
func TestWriterOptions(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().
		IncludeDTD(false).
		AllowPrefixUndeclarations(true).
		WriteTo(&buf, doc)
	require.NoError(t, err)
	// With the DTD excluded, the DOCTYPE must not appear.
	require.NotContains(t, buf.String(), "<!DOCTYPE")

	buf.Reset()
	err = helium.NewWriter().IncludeDTD(true).WriteTo(&buf, doc)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<!DOCTYPE")

	// EscapeNonASCII path with a non-ASCII text node.
	d2 := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	r := d2.CreateElement("r")
	require.NoError(t, d2.AddChild(r))
	require.NoError(t, r.AppendText([]byte("café")))

	buf.Reset()
	err = helium.NewWriter().EscapeNonASCII(true).WriteTo(&buf, d2)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "&#")
}

// TestGetElementByIDFallback covers the O(n) tree-walk fallback path of
// GetElementByID for an API-built document (no parser ID table).
func TestGetElementByIDFallback(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	child := doc.CreateElement("child")
	xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
	_, err := child.SetAttributeNS("id", "target", xmlNS)
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))

	require.Nil(t, doc.IDTable()) // not populated for API-built docs

	found := doc.GetElementByID("target")
	require.Same(t, child, found)

	require.Nil(t, doc.GetElementByID("missing"))

	// SkipIDs short-circuits resolution.
	doc.SetSkipIDs(true)
	require.Nil(t, doc.GetElementByID("target"))
}

// TestElementDeclAndAttrDeclAccessors covers DTD element/attribute declaration
// accessors by parsing a DTD that declares both.
func TestElementDeclAndAttrDeclAccessors(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)

	dtd := doc.IntSubset()
	require.NotNil(t, dtd)

	edecl, ok := dtd.LookupElement("doc", "")
	require.True(t, ok)
	require.Equal(t, enum.MixedElementType, edecl.DeclType())

	adecls := dtd.AttributesForElement("doc")
	require.NotEmpty(t, adecls)
	adecl := adecls[0]
	require.Equal(t, "doc", adecl.Elem())
	require.NotEqual(t, enum.AttrInvalid, adecl.AType())
}

// TestDTDRemoveElement covers RemoveElement.
func TestDTDRemoveElement(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)

	dtd := doc.IntSubset()
	_, ok := dtd.LookupElement("doc", "")
	require.True(t, ok)

	dtd.RemoveElement("doc", "")
	_, ok = dtd.LookupElement("doc", "")
	require.False(t, ok)
}

// TestWriteStringWithoutDTD verifies WriteString on a programmatically built doc.
func TestWriteStringWithoutDTD(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.AppendText([]byte("text & more")))

	s, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.True(t, strings.Contains(s, "<root>"))
	require.Contains(t, s, "&amp;")
}

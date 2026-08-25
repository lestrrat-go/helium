package helium_test

import (
	"os"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestCopyNodeAdjacentTextMerge pins the one real behavior difference between
// AddChild's child-linking (which merges an adjacent Text-after-Text child,
// node.go's addChild fast path) and a naive no-preflight fast link (which
// does not). A source built with two genuinely separate, adjacent Text
// children — constructed via the explicitly-unsafe UnsafeAppendChild so
// AddChild's own merge never gets a chance to collapse them first — must copy
// with the same text-node shape and content that an AddChild-built copy
// produces.
func TestCopyNodeAdjacentTextMerge(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	t1 := doc.CreateText([]byte("hello "))
	t2 := doc.CreateText([]byte("world"))
	require.NoError(t, helium.UnsafeAppendChild(root, t1))
	require.NoError(t, helium.UnsafeAppendChild(root, t2))

	// Sanity: confirm the source really has two adjacent Text children, not
	// one already-merged node.
	var srcChildren []helium.Node
	for c := range helium.Children(root) {
		srcChildren = append(srcChildren, c)
	}
	require.Len(t, srcChildren, 2, "source must have two separate adjacent Text children")

	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	cp, err := helium.CopyNode(root, dst)
	require.NoError(t, err)
	cpElem, ok := helium.AsNode[*helium.Element](cp)
	require.True(t, ok)

	var got []helium.Node
	for c := range helium.Children(cpElem) {
		got = append(got, c)
	}

	// Reference: build the equivalent tree through ordinary AddChild, which
	// merges the two Text nodes into one, then copy that.
	refDoc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	refRoot, err := refDoc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, refDoc.AddChild(refRoot))
	require.NoError(t, refRoot.AddChild(refDoc.CreateText([]byte("hello "))))
	require.NoError(t, refRoot.AddChild(refDoc.CreateText([]byte("world"))))

	refDst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	refCp, err := helium.CopyNode(refRoot, refDst)
	require.NoError(t, err)
	refCpElem, ok := helium.AsNode[*helium.Element](refCp)
	require.True(t, ok)

	var want []helium.Node
	for c := range helium.Children(refCpElem) {
		want = append(want, c)
	}

	require.Len(t, want, 1, "AddChild-built copy merges adjacent text into one node")
	require.Lenf(t, got, len(want), "CopyNode must reproduce AddChild's adjacent-Text merge; got %d text node(s)", len(got))
	for i := range want {
		require.Equal(t, want[i].Content(), got[i].Content())
	}
}

// TestCopyNodeCyclicSourceTerminates pins that CopyNode still terminates on a
// source whose sibling chain has been corrupted into a ring. copyChildren
// iterates the source via Children, whose own per-list cycle guard bounds a
// corrupt sibling chain; removing AddChild's destination-side preflight must
// not reopen a hang on the SOURCE side.
func TestCopyNodeCyclicSourceTerminates(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	var kids []*helium.Element
	for range 5 {
		c, cerr := doc.CreateElement("c")
		require.NoError(t, cerr)
		require.NoError(t, root.AddChild(c))
		kids = append(kids, c)
	}
	// Corrupt the sibling chain into a ring: the last child's next points back
	// to the first.
	helium.UnsafeSetNextSibling(kids[len(kids)-1], kids[0])

	done := make(chan struct{})
	var cp helium.Node
	var copyErr error
	go func() {
		defer close(done)
		dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		cp, copyErr = helium.CopyNode(root, dst)
	}()

	select {
	case <-done:
		require.NoError(t, copyErr)
		require.NotNil(t, cp)
	case <-time.After(5 * time.Second):
		t.Fatal("CopyNode did not terminate on a cyclic source sibling list")
	}
}

// TestCopyNodeStructuralEquivalence copies a representative document —
// nested elements several levels deep, mixed text/comment/CDATA/PI content,
// attributes, and a predefined-entity reference — and checks the copy
// serializes byte-identically to the source's own serialization, as a broad
// safety net over the copier's linking change.
//
// This deliberately avoids namespaces and
// testdata/libxml2-compat/valid/REC-xml-19980210.xml (the tasklist's
// suggested fixture), both for reasons unrelated to copyChildren's linking
// strategy that T2 changes: CopyDoc/CopyNode's namespace handling
// deliberately OVER-DECLARES (see deepCopyOptions.overDeclareNS's doc
// comment), so a default-namespaced source re-declares "xmlns" on every
// descendant element in the copy by design, which a byte-identical
// comparison would wrongly flag; and REC-xml-19980210.xml declares a custom
// internal general entity ("cellback") used in attribute values —
// copyAttributes stores an attribute's resolved Value() (see its doc
// comment), which flattens such a reference to its expansion text on copy,
// while WriteString(src) reproduces the source's own literal "&cellback;"
// markup, so the two serializations differ by construction.
func TestCopyNodeStructuralEquivalence(t *testing.T) {
	t.Parallel()

	const in = `<?xml version="1.0" encoding="UTF-8"?>
<doc version="3">
  <!-- top-level comment -->
  <section id="s1" kind="intro">
    <title>Overview &amp; Scope</title>
    <p>Some <em>mixed</em> content with <![CDATA[raw <not-a-tag> text]]> inside.</p>
    <?pi-target some data ?>
    <section id="s1.1">
      <section id="s1.1.1">
        <p>Deeply nested paragraph.</p>
      </section>
    </section>
  </section>
  <section id="s2">
    <p/>
    <p>trailing text</p>
  </section>
</doc>`

	src, err := helium.NewParser().Parse(t.Context(), []byte(in))
	require.NoError(t, err)

	orig, err := helium.WriteString(src)
	require.NoError(t, err)

	cp, err := helium.CopyDoc(src)
	require.NoError(t, err)

	copied, err := helium.WriteString(cp)
	require.NoError(t, err)

	require.Equal(t, orig, copied, "a deep copy of a representative document must serialize byte-identically to the source")
}

func TestCopyNode(t *testing.T) {
	t.Parallel()

	// CopyNode across several node types.
	t.Run("variants", func(t *testing.T) {
		src := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

		root, err := src.CreateElement("root")
		require.NoError(t, err)
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
	})

	t.Run("node types", func(t *testing.T) {
		t.Run("element with children and attrs", func(t *testing.T) {
			src := helium.NewDefaultDocument()
			root, err := src.CreateElement("root")
			require.NoError(t, err)
			err = root.SetAttribute("id", "1")
			require.NoError(t, err)
			require.NoError(t, src.AddChild(root))

			child, err := src.CreateElement("child")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(child))
			require.NoError(t, child.AppendText([]byte("hello")))

			dst := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(root, dst)
			require.NoError(t, err)

			elem := copied.(*helium.Element)
			require.Equal(t, "root", elem.LocalName())
			val, ok := elem.GetAttribute("id")
			require.True(t, ok)
			require.Equal(t, "1", val)
			require.NotNil(t, elem.FirstChild())
			require.Equal(t, "child", elem.FirstChild().Name())
			require.Equal(t, "hello", string(elem.FirstChild().Content()))
		})

		t.Run("text node", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			txt := doc.CreateText([]byte("hello"))

			dst := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(txt, dst)
			require.NoError(t, err)
			require.Equal(t, helium.TextNode, copied.Type())
			require.Equal(t, "hello", string(copied.Content()))
		})

		t.Run("comment node", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			c := doc.CreateComment([]byte("a comment"))

			dst := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(c, dst)
			require.NoError(t, err)
			require.Equal(t, helium.CommentNode, copied.Type())
			require.Equal(t, "a comment", string(copied.Content()))
		})

		t.Run("CDATA node", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			cd := doc.CreateCDATASection([]byte("cdata content"))

			dst := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(cd, dst)
			require.NoError(t, err)
			require.Equal(t, helium.CDATASectionNode, copied.Type())
			require.Equal(t, "cdata content", string(copied.Content()))
		})

		t.Run("PI node", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			pi := doc.CreatePI("target", "data")

			dst := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(pi, dst)
			require.NoError(t, err)
			require.Equal(t, helium.ProcessingInstructionNode, copied.Type())
		})

		t.Run("element with namespaces", func(t *testing.T) {
			src := helium.NewDefaultDocument()
			root, err := src.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, root.DeclareNamespace("x", "http://example.com"))
			require.NoError(t, root.SetActiveNamespace("x", "http://example.com"))
			require.NoError(t, src.AddChild(root))

			dst := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(root, dst)
			require.NoError(t, err)

			elem := copied.(*helium.Element)
			require.Equal(t, "http://example.com", elem.URI())
		})

		t.Run("element with inherited default namespace", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<article xmlns="http://docbook.org/ns/docbook"><section xml:id="frag"><title>Tools</title></section></article>`))
			require.NoError(t, err)

			section := doc.GetElementByID("frag")
			require.NotNil(t, section)

			dst := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(section, dst)
			require.NoError(t, err)
			require.NoError(t, dst.AddChild(copied))

			root := dst.DocumentElement()
			require.NotNil(t, root)
			require.Equal(t, "http://docbook.org/ns/docbook", root.URI())

			xml, err := helium.WriteString(dst)
			require.NoError(t, err)
			require.Contains(t, xml, `xmlns="http://docbook.org/ns/docbook"`)
		})
	})
}

func TestCopyDoc(t *testing.T) {
	t.Parallel()

	// CopyDoc reproduces the document-level state a
	// caller relies on — URL, property flags, ID-skip state, and the interned ID
	// table — and that the copy is INDEPENDENT of the source (mutating one never
	// affects the other; no mutable map or DTD is aliased).
	t.Run("complete document state", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root (child*)>
<!ELEMENT child (#PCDATA)>
<!ATTLIST child eid ID #IMPLIED>
]>
<root><child eid="c1">a</child><child eid="c2">b</child></root>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		doc.SetURL("http://example.com/base.xml")
		doc.SetProperties(helium.DocWellFormed | helium.DocDTDValid)

		cp, err := helium.CopyDoc(doc)
		require.NoError(t, err)

		// URL, properties carried over.
		require.Equal(t, "http://example.com/base.xml", cp.URL(), "copy preserves the URL")
		require.Equal(t, doc.Properties(), cp.Properties(), "copy preserves property flags")

		// ID table rebuilt: the copy resolves ids, and to its OWN elements.
		cpc1 := cp.GetElementByID("c1")
		require.NotNil(t, cpc1, "copy resolves c1")
		require.NotSame(t, doc.GetElementByID("c1"), cpc1, "copy's id resolves to the copy's element, not the source's")
		require.Same(t, cp, cpc1.OwnerDocument(), "resolved element is owned by the copy")

		// The interned table is independent: registering an id on the copy must not
		// leak into the source, and vice versa.
		cp.RegisterID("c9", cp.DocumentElement())
		require.Nil(t, doc.GetElementByID("c9"), "copy's ID table is not aliased into the source")

		// SkipIDs carried over and independent.
		skipSrc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		skipSrc.SetSkipIDs(true)
		skipCp, err := helium.CopyDoc(skipSrc)
		require.NoError(t, err)
		require.True(t, skipCp.SkipIDs(), "copy preserves SkipIDs")
		require.Nil(t, skipCp.GetElementByID("c1"), "a SkipIDs copy resolves no ids")
		skipCp.SetSkipIDs(false)
		require.True(t, skipSrc.SkipIDs(), "toggling the copy's SkipIDs does not affect the source")
	})

	// parses a document that has a rich internal DTD subset and
	// deep-copies it, then serializes both, exercising copy.go, copy_core.go and
	// the DTD writer paths.
	t.Run("with a DTD", func(t *testing.T) {
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
	})

	// builds a document whose root element holds every
	// leaf child type (Text, CDATA, Comment, PI, EntityRef), then deep-copies it via
	// CopyDoc so the per-node-type branches of the deep copier's copyNode are all
	// exercised.
	t.Run("with mixed children", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
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
	})

	// the deep-copy attribute path with
	// values that the parser has already entity-resolved. Copying such values with
	// the value-PARSING setters (SetParsedAttribute/SetParsedAttributeNS) would
	// re-interpret a bare '&'/'<' — raising "entity was unterminated" for a value
	// that came from '&amp;', and silently double-resolving '&amp;amp;'. The literal
	// setters (SetAttribute/SetAttributeNS) store the resolved value as-is and let
	// the serializer re-escape it, so a CopyDoc round-trips byte-for-byte identically.
	t.Run("entity-bearing attributes", func(t *testing.T) {
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
		// literally, with no re-parse.
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
	})

	t.Run("document tree", func(t *testing.T) {
		t.Run("document with children", func(t *testing.T) {
			src := helium.NewDefaultDocument()
			root, err := src.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, src.AddChild(root))
			require.NoError(t, root.AppendText([]byte("hello")))

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)
			require.NotNil(t, dst)
			require.Equal(t, src.Version(), dst.Version())

			dstRoot := dst.DocumentElement()
			require.NotNil(t, dstRoot)
			require.Equal(t, "root", dstRoot.LocalName())
			require.Equal(t, "hello", string(dstRoot.Content()))
		})

		t.Run("line numbers preserved on elements and attributes", func(t *testing.T) {
			// A faithful deep copy must carry source line numbers onto every copied
			// node — including ATTRIBUTES — so diagnostics emitted against a copied tree
			// (e.g. xsd's conditional-inclusion clone) keep their source locations.
			src := helium.NewDefaultDocument()
			root, err := src.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, src.AddChild(root))
			root.SetLine(7)

			err = root.SetAttribute("a", "v")
			require.NoError(t, err)
			plainAttr, ok := root.FindAttribute(helium.NSPredicate{Local: "a", NamespaceURI: ""})
			require.True(t, ok)
			plainAttr.SetLine(42)

			ns, err := src.CreateNamespace("p", "urn:p")
			require.NoError(t, err)
			err = root.SetAttributeNS("b", "w", ns)
			require.NoError(t, err)
			nsAttr, ok := root.FindAttribute(helium.NSPredicate{Local: "b", NamespaceURI: "urn:p"})
			require.True(t, ok)
			nsAttr.SetLine(99)

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)
			dstRoot := dst.DocumentElement()
			require.NotNil(t, dstRoot)
			require.Equal(t, 7, dstRoot.Line())

			dstPlain, ok := dstRoot.FindAttribute(helium.NSPredicate{Local: "a", NamespaceURI: ""})
			require.True(t, ok)
			require.Equal(t, 42, dstPlain.Line())

			dstNS, ok := dstRoot.FindAttribute(helium.NSPredicate{Local: "b", NamespaceURI: "urn:p"})
			require.True(t, ok)
			require.Equal(t, 99, dstNS.Line())
		})

		t.Run("document with DTD", func(t *testing.T) {
			src := helium.NewDefaultDocument()
			_, err := src.CreateInternalSubset("root", "", "root.dtd")
			require.NoError(t, err)

			root, err := src.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, src.AddChild(root))

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)
			require.NotNil(t, dst.IntSubset())
			require.Equal(t, "root", dst.IntSubset().Name())
		})

		t.Run("DTD entities copied", func(t *testing.T) {
			src := helium.NewDefaultDocument()
			dtd, err := src.CreateInternalSubset("root", "", "")
			require.NoError(t, err)

			_, err = dtd.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
			require.NoError(t, err)
			_, err = dtd.AddEntity("baz", enum.InternalGeneralEntity, "", "", "qux")
			require.NoError(t, err)

			root, err := src.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, src.AddChild(root))

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)

			dstDTD := dst.IntSubset()
			require.NotNil(t, dstDTD)

			ent, ok := dstDTD.LookupEntity("foo")
			require.True(t, ok)
			require.Equal(t, "bar", string(ent.Content()))

			ent, ok = dstDTD.LookupEntity("baz")
			require.True(t, ok)
			require.Equal(t, "qux", string(ent.Content()))

			// Verify independence: mutating src doesn't affect dst.
			srcEnt, _ := src.IntSubset().LookupEntity("foo")
			require.NotSame(t, srcEnt, ent)
		})

		t.Run("DTD element declarations copied", func(t *testing.T) {
			const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (child)>
  <!ELEMENT child (#PCDATA)>
]>
<root><child>text</child></root>`

			p := helium.NewParser()
			src, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)

			dstDTD := dst.IntSubset()
			require.NotNil(t, dstDTD)

			edecl, ok := dstDTD.LookupElement("root", "")
			require.True(t, ok)
			require.Equal(t, "root", edecl.Name())

			edecl, ok = dstDTD.LookupElement("child", "")
			require.True(t, ok)
			require.Equal(t, "child", edecl.Name())
		})

		t.Run("DTD attribute declarations copied", func(t *testing.T) {
			const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root id ID #IMPLIED>
  <!ATTLIST root class CDATA "default">
]>
<root id="x"/>`

			p := helium.NewParser()
			src, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)

			dstDTD := dst.IntSubset()
			require.NotNil(t, dstDTD)

			adecl, ok := dstDTD.LookupAttribute("id", "", "root")
			require.True(t, ok)
			require.Equal(t, enum.AttrID, adecl.AType())

			adecl, ok = dstDTD.LookupAttribute("class", "", "root")
			require.True(t, ok)
			require.Equal(t, enum.AttrCDATA, adecl.AType())
		})

		t.Run("DTD notations copied", func(t *testing.T) {
			src := helium.NewDefaultDocument()
			dtd, err := src.CreateInternalSubset("root", "", "")
			require.NoError(t, err)

			_, err = dtd.AddNotation("gif", "image/gif", "")
			require.NoError(t, err)

			root, err := src.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, src.AddChild(root))

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)

			dstDTD := dst.IntSubset()
			require.NotNil(t, dstDTD)

			nota, ok := dstDTD.LookupNotation("gif")
			require.True(t, ok)
			require.Equal(t, "gif", nota.Name())
		})

		t.Run("DTD parameter entities copied", func(t *testing.T) {
			src := helium.NewDefaultDocument()
			dtd, err := src.CreateInternalSubset("root", "", "")
			require.NoError(t, err)

			_, err = dtd.AddEntity("pe", enum.InternalParameterEntity, "", "", "param-content")
			require.NoError(t, err)

			root, err := src.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, src.AddChild(root))

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)

			dstDTD := dst.IntSubset()
			require.NotNil(t, dstDTD)

			pent, ok := dstDTD.LookupParameterEntity("pe")
			require.True(t, ok)
			require.Equal(t, "param-content", string(pent.Content()))
		})

		t.Run("copied DTD serializes correctly", func(t *testing.T) {
			const input = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root (child)>
<!ELEMENT child (#PCDATA)>
<!ENTITY foo "bar">
<!ATTLIST root id ID #IMPLIED>
]>
<root id="x"><child>text</child></root>`

			p := helium.NewParser()
			src, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			srcXML, err := helium.WriteString(src)
			require.NoError(t, err)

			dst, err := helium.CopyDoc(src)
			require.NoError(t, err)

			dstXML, err := helium.WriteString(dst)
			require.NoError(t, err)

			require.Equal(t, srcXML, dstXML)
		})

		t.Run("nil document", func(t *testing.T) {
			_, err := helium.CopyDoc(nil)
			require.Error(t, err)
		})
	})
}

func TestCopyDTD(t *testing.T) {
	t.Parallel()

	// copies the internal-subset DTD declarations from one document
	// into another via CopyDTDInfo.
	t.Run("internal subset", func(t *testing.T) {
		in, err := os.ReadFile("test/att12.xml")
		require.NoError(t, err)
		src, err := helium.NewParser().Parse(t.Context(), in)
		require.NoError(t, err)
		require.NotNil(t, src.IntSubset())

		dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		require.NoError(t, helium.CopyDTDInfo(src, dst))

		require.NotNil(t, dst.IntSubset(), "CopyDTDInfo populates the destination internal subset")
		_, ok := dst.IntSubset().LookupNotation("gif")
		require.True(t, ok, "notation copied via CopyDTDInfo")

		// nil arguments are a no-op (no panic, no error).
		require.NoError(t, helium.CopyDTDInfo(nil, dst))
		require.NoError(t, helium.CopyDTDInfo(src, nil))

		// A destination that already has an internal subset surfaces the error
		// instead of silently discarding the copy.
		occupied := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		_, err = occupied.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		require.Error(t, helium.CopyDTDInfo(src, occupied),
			"CopyDTDInfo surfaces the dest-already-has-subset error")
	})

	// copies an external DTD subset between documents.
	t.Run("external subset", func(t *testing.T) {
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
	})
}

// attrDeclNames returns the names of the attribute declarations in decls, so a
// mismatch reads as a name list in the failure output.
func attrDeclNames(decls []*helium.AttributeDecl) []string {
	names := make([]string, len(decls))
	for i, d := range decls {
		names[i] = d.Name()
	}
	return names
}

// dtdAttrDecls collects the attribute declarations linked in dtd's child list,
// in child-list order.
func dtdAttrDecls(dtd *helium.DTD) []*helium.AttributeDecl {
	var decls []*helium.AttributeDecl
	for c := range helium.Children(dtd) {
		if adecl, ok := helium.AsNode[*helium.AttributeDecl](c); ok {
			decls = append(decls, adecl)
		}
	}
	return decls
}

// A DTD records its attribute declarations in registration order (the <!ATTLIST>
// order), which is what AttributesForElement reports; the DTD child list is a
// separate, independently mutable ordering that drives serialization. Relinking
// a declaration (AttributeDecl.AddSibling) moves it in the child list WITHOUT
// re-registering it, so the two orders come apart. A deep copy must reproduce
// the source's registration order, not the child-list order it happens to walk.
func TestCopyDTDAttrDeclOrder(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root a1 CDATA #IMPLIED>
<!ATTLIST root a2 CDATA #IMPLIED>
<!ATTLIST root a3 CDATA #IMPLIED>
<!ATTLIST root a4 CDATA #IMPLIED>
]>
<root/>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	dtd := doc.IntSubset()
	require.NotNil(t, dtd)

	linked := dtdAttrDecls(dtd)
	require.Equal(t, []string{"a1", "a2", "a3", "a4"}, attrDeclNames(linked))

	// Move a1 to the end of the child list. Registration order is untouched.
	require.NoError(t, linked[3].AddSibling(linked[0]))
	require.Equal(t, []string{"a2", "a3", "a4", "a1"}, attrDeclNames(dtdAttrDecls(dtd)),
		"the child list must now differ from registration order for this test to mean anything")
	require.Equal(t, []string{"a1", "a2", "a3", "a4"}, attrDeclNames(dtd.AttributesForElement("root")),
		"relinking a declaration must not change registration order")

	cp, err := helium.CopyDoc(doc)
	require.NoError(t, err)
	cpDTD := cp.IntSubset()
	require.NotNil(t, cpDTD)

	require.Equal(t, attrDeclNames(dtd.AttributesForElement("root")), attrDeclNames(cpDTD.AttributesForElement("root")),
		"a copied DTD must report the source's registration order")
	require.Equal(t, []string{"a2", "a3", "a4", "a1"}, attrDeclNames(dtdAttrDecls(cpDTD)),
		"the copy's child list still follows the source's child list, so serialization round-trips")
}

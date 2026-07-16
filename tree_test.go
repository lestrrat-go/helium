package helium_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestDocumentElement(t *testing.T) {
	t.Run("with element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(e))

		got := doc.DocumentElement()
		require.Equal(t, e, got)
	})

	t.Run("without element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		got := doc.DocumentElement()
		require.Nil(t, got)
	})

	t.Run("PI before element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		pi := doc.CreatePI("target", "data")
		require.NoError(t, doc.AddChild(pi))

		e := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(e))

		got := doc.DocumentElement()
		require.Equal(t, e, got)
	})
}

func TestSetDocumentElement(t *testing.T) {
	t.Run("literal nil returns ErrNilNode", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		err := doc.SetDocumentElement(nil)
		require.ErrorIs(t, err, helium.ErrNilNode)
	})

	t.Run("typed nil returns ErrNilNode without panic", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		var root *helium.Element
		err := doc.SetDocumentElement(root)
		require.ErrorIs(t, err, helium.ErrNilNode)
	})

	t.Run("document self-insertion is rejected and leaves doc untouched", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		err := doc.SetDocumentElement(doc)
		require.Error(t, err)
		require.Nil(t, doc.Parent(), "rejected insertion must not link the candidate")
		requireNoCycle(t, doc)
	})

	t.Run("non-element node is rejected and leaves doc untouched", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			node helium.MutableNode
		}{
			{"text node", helium.NewDefaultDocument().CreateText([]byte("x"))},
			{"comment node", helium.NewDefaultDocument().CreateComment([]byte("c"))},
		} {
			t.Run(tc.name, func(t *testing.T) {
				doc := helium.NewDefaultDocument()
				err := doc.SetDocumentElement(tc.node)
				require.ErrorIs(t, err, helium.ErrInvalidOperation)
				require.Nil(t, doc.FirstChild(), "rejected non-element must not be linked as a child")
				require.Nil(t, doc.DocumentElement(), "doc has no document element")
			})
		}
	})

	t.Run("element-kind marker that is not a concrete *Element is rejected", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		// An XIncludeMarker reports ElementNode but is not a real *Element, so it
		// must not become the document element — DocumentElement() would never
		// return it, leaving the document element effectively nil.
		marker := helium.NewXIncludeMarker(doc, helium.ElementNode, "fake-element")
		err := doc.SetDocumentElement(marker)
		require.ErrorIs(t, err, helium.ErrInvalidOperation)
		require.Nil(t, doc.FirstChild(), "spoofed-kind marker must not be linked as a child")
		require.Nil(t, doc.DocumentElement(), "doc has no document element")
	})

	t.Run("nil receiver returns ErrNilNode", func(t *testing.T) {
		var doc *helium.Document
		err := doc.SetDocumentElement(helium.NewDefaultDocument().CreateElement("root"))
		require.ErrorIs(t, err, helium.ErrNilNode)
	})

	t.Run("replace document element with existing descendant", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))

		child := doc.CreateElement("child")
		require.NoError(t, root.AddChild(child))

		require.NoError(t, doc.SetDocumentElement(child))
		require.Equal(t, helium.Node(child), doc.DocumentElement())
		require.Nil(t, root.Parent())
		requireNoCycle(t, doc)
		requireNoCycle(t, child)
	})
}

// TestRawLinkageBehindUnsafeSurface documents that raw single-pointer linkage
// is only reachable through the explicitly-unsafe UnsafeSet* functions, while
// the ordinary guarded path (AddChild) rejects the same cycle.
func TestRawLinkageBehindUnsafeSurface(t *testing.T) {
	doc := helium.NewDefaultDocument()
	a := doc.CreateElement("a")
	b := doc.CreateElement("b")
	require.NoError(t, a.AddChild(b))

	// The guarded path refuses to form a parent cycle.
	require.Error(t, b.AddChild(a), "AddChild must reject a cycle")

	// The unsafe primitive still builds one when a caller explicitly opts in.
	helium.UnsafeSetParent(a, b)
	require.Equal(t, helium.Node(b), a.Parent())
}

func TestUnlinkNode(t *testing.T) {
	t.Run("unlink middle child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent := doc.CreateElement("parent")
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		c := doc.CreateElement("c")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))
		require.NoError(t, parent.AddChild(c))

		helium.UnlinkNode(b)

		require.Nil(t, b.Parent())
		require.Nil(t, b.PrevSibling())
		require.Nil(t, b.NextSibling())
		require.Equal(t, c, a.NextSibling())
		require.Equal(t, a, c.PrevSibling())
	})

	t.Run("unlink first child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent := doc.CreateElement("parent")
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		helium.UnlinkNode(a)

		require.Equal(t, helium.Node(b), parent.FirstChild())
		require.Nil(t, b.PrevSibling())
		require.Nil(t, a.Parent())
	})

	t.Run("unlink last child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent := doc.CreateElement("parent")
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		helium.UnlinkNode(b)

		require.Equal(t, helium.Node(a), parent.LastChild())
		require.Nil(t, a.NextSibling())
		require.Nil(t, b.Parent())
	})

	t.Run("unlink only child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent := doc.CreateElement("parent")
		a := doc.CreateElement("a")
		require.NoError(t, parent.AddChild(a))

		helium.UnlinkNode(a)

		require.Nil(t, parent.FirstChild())
		require.Nil(t, parent.LastChild())
		require.Nil(t, a.Parent())
	})

	t.Run("unlink nil is no-op", func(t *testing.T) {
		helium.UnlinkNode(nil) // should not panic
	})
}

func TestWalk(t *testing.T) {
	t.Run("sees sibling replacement during traversal", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		a := doc.CreateElement("a")
		c := doc.CreateElement("c")

		require.NoError(t, doc.AddChild(root))
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(c))

		var visited []string
		err := helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
			if n.Type() != helium.ElementNode {
				return nil
			}

			visited = append(visited, n.Name())
			if n == a {
				b := doc.CreateElement("b")
				require.NoError(t, c.Replace(b))
			}
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, []string{"root", "a", "b"}, visited)
	})

	t.Run("skips sibling removed during traversal", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		a := doc.CreateElement("a")
		c := doc.CreateElement("c")

		require.NoError(t, doc.AddChild(root))
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(c))

		var visited []string
		err := helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
			if n.Type() != helium.ElementNode {
				return nil
			}

			visited = append(visited, n.Name())
			if n == a {
				helium.UnlinkNode(c)
			}
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, []string{"root", "a"}, visited)
	})
}

func TestText(t *testing.T) {
	t.Run("AppendText", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		n := doc.CreateText([]byte("Hello "))
		require.NoError(t, n.AppendText([]byte("World!")), "AppendText succeeds")
		require.Equal(t, []byte("Hello World!"), n.Content(), "Content matches")
	})

	t.Run("AddChild merges text", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		n1 := doc.CreateText([]byte("Hello "))
		n2 := doc.CreateText([]byte("World!"))

		require.NoError(t, n1.AddChild(n2), "AddChild succeeds")
		require.Equal(t, []byte("Hello World!"), n1.Content(), "Content matches")
	})

	t.Run("AddChild rejects non-text", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		n1 := doc.CreateText([]byte("Hello "))
		n2 := &helium.ProcessingInstruction{}

		require.Equal(t, helium.ErrInvalidOperation, n1.AddChild(n2), "AddChild fails")
		require.Equal(t, []byte("Hello "), n1.Content(), "Content matches")
	})
}

func TestLookupNSByHref(t *testing.T) {
	t.Run("found on element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		require.NoError(t, e.DeclareNamespace("x", "http://example.com"))

		ns := helium.LookupNSByHref(e, "http://example.com")
		require.NotNil(t, ns)
		require.Equal(t, "x", ns.Prefix())
	})

	t.Run("found on ancestor", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent := doc.CreateElement("parent")
		require.NoError(t, parent.DeclareNamespace("x", "http://example.com"))

		child := doc.CreateElement("child")
		require.NoError(t, parent.AddChild(child))

		ns := helium.LookupNSByHref(child, "http://example.com")
		require.NotNil(t, ns)
		require.Equal(t, "x", ns.Prefix())
	})

	t.Run("xml namespace", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns := helium.LookupNSByHref(e, lexicon.NamespaceXML)
		require.NotNil(t, ns)
		require.Equal(t, "xml", ns.Prefix())
	})

	t.Run("not found", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns := helium.LookupNSByHref(e, "http://not.found.com")
		require.Nil(t, ns)
	})
}

func TestLookupNSByPrefix(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e := doc.CreateElement("root")
	require.NoError(t, e.DeclareNamespace("x", "http://example.com"))

	ns := helium.LookupNSByPrefix(e, "x")
	require.NotNil(t, ns)
	require.Equal(t, "http://example.com", ns.URI())

	ns = helium.LookupNSByPrefix(e, "xml")
	require.NotNil(t, ns)
	require.Equal(t, lexicon.NamespaceXML, ns.URI())

	ns = helium.LookupNSByPrefix(e, "missing")
	require.Nil(t, ns)
}

func TestNodeGetBase(t *testing.T) {
	t.Run("no base", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(e))

		base := helium.NodeGetBase(doc, e)
		require.Equal(t, "", base)
	})

	t.Run("direct xml:base", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(e))

		xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
		_, err := e.SetAttributeNS("base", "http://example.com/", xmlNS)
		require.NoError(t, err)

		base := helium.NodeGetBase(doc, e)
		require.Equal(t, "http://example.com/", base)
	})

	t.Run("inherited xml:base", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent := doc.CreateElement("parent")
		require.NoError(t, doc.AddChild(parent))

		xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
		_, err := parent.SetAttributeNS("base", "http://example.com/dir/", xmlNS)
		require.NoError(t, err)

		child := doc.CreateElement("child")
		require.NoError(t, parent.AddChild(child))

		base := helium.NodeGetBase(doc, child)
		require.Equal(t, "http://example.com/dir/", base)
	})

	t.Run("relative resolution", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent := doc.CreateElement("parent")
		require.NoError(t, doc.AddChild(parent))

		xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
		_, err := parent.SetAttributeNS("base", "http://example.com/a/b/", xmlNS)
		require.NoError(t, err)

		child := doc.CreateElement("child")
		require.NoError(t, parent.AddChild(child))
		_, err = child.SetAttributeNS("base", "c/d/", xmlNS)
		require.NoError(t, err)

		base := helium.NodeGetBase(doc, child)
		require.Equal(t, "http://example.com/a/b/c/d/", base)
	})
}

func TestDocumentURL(t *testing.T) {
	t.Run("set and get URL", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		require.Equal(t, "", doc.URL())

		doc.SetURL("http://example.com/doc.xml")
		require.Equal(t, "http://example.com/doc.xml", doc.URL())
	})

	t.Run("URL used as base in NodeGetBase", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		doc.SetURL("http://example.com/dir/doc.xml")

		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))

		base := helium.NodeGetBase(doc, root)
		require.Equal(t, "http://example.com/dir/doc.xml", base)
	})

	t.Run("URL with relative xml:base", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		doc.SetURL("http://example.com/dir/doc.xml")

		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))

		xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
		_, err := root.SetAttributeNS("base", "sub/", xmlNS)
		require.NoError(t, err)

		base := helium.NodeGetBase(doc, root)
		require.Equal(t, "http://example.com/dir/sub/", base)
	})

	t.Run("URL set during parsing", func(t *testing.T) {
		const input = `<?xml version="1.0"?><root/>`
		p := helium.NewParser().BaseURI("/some/path/doc.xml")
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.Equal(t, "/some/path/doc.xml", doc.URL())
	})
}

func TestCopyNode(t *testing.T) {
	t.Run("element with children and attrs", func(t *testing.T) {
		src := helium.NewDefaultDocument()
		root := src.CreateElement("root")
		_, err := root.SetAttribute("id", "1")
		require.NoError(t, err)
		require.NoError(t, src.AddChild(root))

		child := src.CreateElement("child")
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
		root := src.CreateElement("root")
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
}

func TestCopyDoc(t *testing.T) {
	t.Run("document with children", func(t *testing.T) {
		src := helium.NewDefaultDocument()
		root := src.CreateElement("root")
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
		root := src.CreateElement("root")
		require.NoError(t, src.AddChild(root))
		root.SetLine(7)

		_, err := root.SetAttribute("a", "v")
		require.NoError(t, err)
		plainAttr, ok := root.FindAttribute(helium.NSPredicate{Local: "a", NamespaceURI: ""})
		require.True(t, ok)
		plainAttr.SetLine(42)

		ns, err := src.CreateNamespace("p", "urn:p")
		require.NoError(t, err)
		_, err = root.SetAttributeNS("b", "w", ns)
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

		root := src.CreateElement("root")
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

		root := src.CreateElement("root")
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

		root := src.CreateElement("root")
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

		root := src.CreateElement("root")
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
}

func TestReplaceDetachesOldNode(t *testing.T) {
	// Build <root><a/><secret/><b/></root>
	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	a := doc.CreateElement("a")
	secret := doc.CreateElement("secret")
	b := doc.CreateElement("b")
	require.NoError(t, root.AddChild(a))
	require.NoError(t, root.AddChild(secret))
	require.NoError(t, root.AddChild(b))

	repl := doc.CreateElement("EncryptedData")
	require.NoError(t, secret.Replace(repl))

	// After replacement the old node must be fully detached.
	require.Nil(t, secret.Parent(), "replaced node parent must be cleared")
	require.Nil(t, secret.PrevSibling(), "replaced node prev must be cleared")
	require.Nil(t, secret.NextSibling(), "replaced node next must be cleared")

	// Tree must read a / EncryptedData / b.
	require.Equal(t, a, root.FirstChild())
	require.Equal(t, repl, a.NextSibling())
	require.Equal(t, b, repl.NextSibling())
	require.Equal(t, b, root.LastChild())

	// A stale UnlinkNode on the old handle must NOT corrupt the tree.
	helium.UnlinkNode(secret)
	require.Equal(t, a, root.FirstChild())
	require.Equal(t, repl, a.NextSibling())
	require.Equal(t, b, repl.NextSibling())
	require.Equal(t, b, root.LastChild())
	require.Equal(t, repl, b.PrevSibling())
}

func TestReplaceSelf(t *testing.T) {
	t.Run("exact self-replacement is a no-op", func(t *testing.T) {
		// Build <root><a/><b/></root>
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(b))

		// Replacing a node with itself must leave the tree intact.
		require.NoError(t, a.Replace(a))

		require.Equal(t, root, a.Parent(), "a.Parent() must remain root")
		require.Equal(t, b, a.NextSibling(), "a.NextSibling() must remain b")
		require.Equal(t, a, root.FirstChild(), "root.FirstChild() must remain a")
		require.Equal(t, b, root.LastChild())
		require.Equal(t, a, b.PrevSibling())
	})

	t.Run("replacement list including the replaced node keeps it live", func(t *testing.T) {
		// Build <root><a/><b/></root>
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(b))

		// Replace a with [a, c]: a stays live, c is inserted after it.
		c := doc.CreateElement("c")
		require.NoError(t, a.Replace(a, c))

		require.Equal(t, root, a.Parent())
		require.Equal(t, a, root.FirstChild())
		require.Equal(t, c, a.NextSibling())
		require.Equal(t, b, c.NextSibling())
		require.Equal(t, b, root.LastChild())
	})
}

func TestReplaceWithExistingSibling(t *testing.T) {
	t.Run("replace node with its own next sibling", func(t *testing.T) {
		// Build <root><a/><b/></root>
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(b))

		// Replacing a with its own next sibling b must yield a
		// well-formed chain with just b as root's only child.
		require.NoError(t, a.Replace(b))

		require.Equal(t, root, b.Parent())
		require.Equal(t, b, root.FirstChild())
		require.Equal(t, b, root.LastChild())
		require.Nil(t, b.NextSibling(), "b.NextSibling() must be nil (no self-loop)")
		require.Nil(t, b.PrevSibling(), "b.PrevSibling() must be nil (no self-loop)")
	})

	t.Run("replace node with its own previous sibling", func(t *testing.T) {
		// Build <root><a/><b/></root>
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(b))

		// Replacing b with its own previous sibling a must yield a
		// well-formed chain with just a as root's only child.
		require.NoError(t, b.Replace(a))

		require.Equal(t, root, a.Parent())
		require.Equal(t, a, root.FirstChild())
		require.Equal(t, a, root.LastChild())
		require.Nil(t, a.NextSibling(), "a.NextSibling() must be nil (no self-loop)")
		require.Nil(t, a.PrevSibling(), "a.PrevSibling() must be nil (no self-loop)")
	})

	t.Run("replace middle node with its next sibling", func(t *testing.T) {
		// Build <root><a/><b/><c/></root>
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		c := doc.CreateElement("c")
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(b))
		require.NoError(t, root.AddChild(c))

		// Replace b with c: chain becomes a / c with no self-loop.
		require.NoError(t, b.Replace(c))

		require.Equal(t, a, root.FirstChild())
		require.Equal(t, c, a.NextSibling())
		require.Equal(t, a, c.PrevSibling())
		require.Nil(t, c.NextSibling(), "c.NextSibling() must be nil")
		require.Equal(t, c, root.LastChild())
	})
}

// TestUnsafeAppendChild covers the public UnsafeAppendChild helper across the
// empty-parent and non-empty-parent fast paths.
func TestUnsafeAppendChild(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	parent := doc.CreateElement("parent")

	first := doc.CreateElement("first")
	require.NoError(t, helium.UnsafeAppendChild(parent, first), "fast-link first child")
	require.Equal(t, helium.Node(first), parent.FirstChild())
	require.Equal(t, helium.Node(first), parent.LastChild())
	require.Equal(t, helium.Node(parent), first.Parent())

	second := doc.CreateElement("second")
	require.NoError(t, helium.UnsafeAppendChild(parent, second), "fast-link second child")
	require.Equal(t, helium.Node(second), parent.LastChild())
	require.Equal(t, helium.Node(second), first.NextSibling())
	require.Equal(t, helium.Node(first), second.PrevSibling())
}

// TestNodeLine covers docnode.Line via a parsed node that carries line info.
func TestNodeLine(t *testing.T) {
	t.Parallel()
	const src = "<root>\n  <child/>\n</root>"
	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)
	// Line() returns the recorded line number; it must be a non-negative int and
	// not panic. We assert it is callable and consistent.
	require.GreaterOrEqual(t, root.Line(), 0)
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			require.GreaterOrEqual(t, c.Line(), 0)
		}
	}
}

// TestDocumentAppendText covers Document.AppendText, which appends a Text child
// to the document, merging into a trailing Text node when possible.
func TestDocumentAppendText(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	require.NoError(t, doc.AppendText([]byte("hello")))
	require.NoError(t, doc.AppendText([]byte(" world")))

	var found bool
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.TextNode {
			found = true
			require.Contains(t, string(c.Content()), "hello")
		}
	}
	require.True(t, found, "document gained a text child")
}

// TestClarkName covers the ClarkName helper.
func TestClarkName(t *testing.T) {
	t.Parallel()
	require.Equal(t, "{urn:example}local", helium.ClarkName("urn:example", "local"))
	require.Equal(t, "{}local", helium.ClarkName("", "local"))
}

// TestNamespaceNodeWrapperContent covers NamespaceNodeWrapper.Content.
func TestNamespaceNodeWrapperContent(t *testing.T) {
	t.Parallel()
	ns := helium.NewNamespace("p", "urn:example")
	nsw := helium.NewNamespaceNodeWrapper(ns, nil)
	require.Equal(t, "urn:example", string(nsw.Content()))
	require.Equal(t, "p", nsw.Name())
}

// TestNodeNamespaceMethods covers DeclareNamespace, SetActiveNamespace, SetNs,
// AddNamespaceDecl and the qname caching in Name().
func TestNodeNamespaceMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.DeclareNamespace("p", "http://example.com/p"))
	require.NoError(t, root.SetActiveNamespace("p", "http://example.com/p"))

	// Name() now reflects the prefix and caches the qname.
	require.Equal(t, "p:root", root.Name())
	require.Equal(t, "p:root", root.Name()) // cached path
	require.Equal(t, "p", root.Prefix())
	require.Equal(t, "http://example.com/p", root.URI())

	// AddNamespaceDecl with an existing namespace object.
	ns := helium.NewNamespace("q", "http://example.com/q")
	root.AddNamespaceDecl(ns)
	root.SetNs(ns)
	require.Equal(t, "q:root", root.Name())
}

// TestBuildURI exercises BuildURI across local-path, http, and absolute cases.
func TestBuildURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		systemID string
		base     string
		want     string
	}{
		{"absolute system id is returned verbatim", "http://x/a.dtd", "http://y/", "http://x/a.dtd"},
		{"relative against http base", "a.dtd", "http://host/dir/doc.xml", "http://host/dir/a.dtd"},
		{"relative against file path", "a.dtd", "/dir/doc.xml", "/dir/a.dtd"},
		{"absolute local path", "/abs/a.dtd", "/dir/doc.xml", "/abs/a.dtd"},
		// Windows shapes are plain strings, so the Windows behavior below is
		// exercised on any GOOS. A native Windows base must NOT route the drive
		// letter through url.Parse (which would emit "c:///a.dtd"); it resolves
		// with local-path (forward-slash) semantics.
		{"relative against windows backslash base", "child.xml", `C:\dir\main.xml`, "C:/dir/child.xml"},
		{"relative against windows forward-slash base", "a.dtd", "D:/dir/doc.xml", "D:/dir/a.dtd"},
		{"windows-absolute system id returned verbatim", `C:\abs\a.dtd`, `D:\dir\doc.xml`, `C:\abs\a.dtd`},
		{"interior dot-dot against windows base", "../sib/child.xml", `C:\a\b\main.xml`, "C:/a/sib/child.xml"},
		{"unc base resolves relative ref", "child.xml", `\\host\share\main.xml`, "//host/share/child.xml"},
		// An absolute-URI systemID stands on its own even when the base is a
		// native Windows path. Without the scheme check this collapsed "http://"
		// to "http:/" and joined it onto the drive-letter base (Windows-only
		// regression that broke the W3C resolve-uri/base-uri cluster).
		{"absolute http system id against windows drive base", "http://example.com/a/b", `D:\dir\doc.xsl`, "http://example.com/a/b"},
		{"absolute http system id against windows slash base", "http://example.com/a/b", "D:/dir/doc.xsl", "http://example.com/a/b"},
		{"absolute file system id against windows base", "file:///x/y", `C:\dir\doc.xsl`, "file:///x/y"},
		// A RELATIVE Windows base (backslashes, no drive — what filepath.Join
		// yields on Windows for a relative test path) must keep its directory so a
		// sibling entity resolves inside it. Without backslash-aware handling this
		// dropped to a bare "world.txt" and the external entity could not be found.
		{"sibling against relative windows base", "world.txt", `..\d\e\example.xml`, "../d/e/world.txt"},
		// A file: base with a Windows drive letter must yield a proper file: URI
		// (not the drive-rooted "/D:/..." path url.Parse exposes), so file-URI-aware
		// loaders convert it back to a native path. The POSIX file: base below
		// keeps returning a plain path, proving POSIX is unaffected.
		{"sibling against windows drive file uri", "nested.dtd", "file:///D:/tmp/t/inc.xml", "file:///D:/tmp/t/nested.dtd"},
		{"sibling against posix file uri", "nested.dtd", "file:///tmp/t/inc.xml", "/tmp/t/nested.dtd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, helium.BuildURI(tt.systemID, tt.base))
		})
	}
}

// TestNodeGetBaseAndSet exercises NodeGetBase with xml:base attributes and the
// SetNodeBaseURI override.
func TestNodeGetBaseAndSet(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	doc.SetURL("http://example.com/dir/doc.xml")

	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	child := doc.CreateElement("child")
	xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
	require.NoError(t, child.SetLiteralAttributeNS("base", "sub/", xmlNS))
	require.NoError(t, root.AddChild(child))

	// The child's effective base resolves its xml:base against the doc URL.
	base := helium.NodeGetBase(doc, child)
	require.Contains(t, base, "sub")

	// A nil node yields an empty base.
	require.Equal(t, "", helium.NodeGetBase(doc, nil))

	// SetNodeBaseURI installs an explicit entity base URI that takes precedence.
	helium.SetNodeBaseURI(child, "http://other.example/")
	base = helium.NodeGetBase(doc, child)
	require.Contains(t, base, "other.example")
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

package helium_test

import (
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

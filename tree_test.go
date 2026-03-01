package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentElement(t *testing.T) {
	t.Run("with element", func(t *testing.T) {
		doc := CreateDocument()
		e, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(e))

		got := doc.DocumentElement()
		require.Equal(t, e, got)
	})

	t.Run("without element", func(t *testing.T) {
		doc := CreateDocument()
		got := doc.DocumentElement()
		require.Nil(t, got)
	})

	t.Run("PI before element", func(t *testing.T) {
		doc := CreateDocument()
		pi, err := doc.CreatePI("target", "data")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(pi))

		e, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(e))

		got := doc.DocumentElement()
		require.Equal(t, e, got)
	})
}

func TestUnlinkNode(t *testing.T) {
	t.Run("unlink middle child", func(t *testing.T) {
		parent := newElement("parent")
		a := newElement("a")
		b := newElement("b")
		c := newElement("c")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))
		require.NoError(t, parent.AddChild(c))

		UnlinkNode(b)

		require.Nil(t, b.Parent())
		require.Nil(t, b.PrevSibling())
		require.Nil(t, b.NextSibling())
		require.Equal(t, c, a.NextSibling())
		require.Equal(t, a, c.PrevSibling())
	})

	t.Run("unlink first child", func(t *testing.T) {
		parent := newElement("parent")
		a := newElement("a")
		b := newElement("b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		UnlinkNode(a)

		require.Equal(t, Node(b), parent.FirstChild())
		require.Nil(t, b.PrevSibling())
		require.Nil(t, a.Parent())
	})

	t.Run("unlink last child", func(t *testing.T) {
		parent := newElement("parent")
		a := newElement("a")
		b := newElement("b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		UnlinkNode(b)

		require.Equal(t, Node(a), parent.LastChild())
		require.Nil(t, a.NextSibling())
		require.Nil(t, b.Parent())
	})

	t.Run("unlink only child", func(t *testing.T) {
		parent := newElement("parent")
		a := newElement("a")
		require.NoError(t, parent.AddChild(a))

		UnlinkNode(a)

		require.Nil(t, parent.FirstChild())
		require.Nil(t, parent.LastChild())
		require.Nil(t, a.Parent())
	})

	t.Run("unlink nil is no-op", func(t *testing.T) {
		UnlinkNode(nil) // should not panic
	})
}

func TestLookupNSByHref(t *testing.T) {
	t.Run("found on element", func(t *testing.T) {
		doc := CreateDocument()
		e, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, e.SetNamespace("x", "http://example.com"))

		ns := LookupNSByHref(e, "http://example.com")
		require.NotNil(t, ns)
		require.Equal(t, "x", ns.Prefix())
	})

	t.Run("found on ancestor", func(t *testing.T) {
		doc := CreateDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		require.NoError(t, parent.SetNamespace("x", "http://example.com"))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))

		ns := LookupNSByHref(child, "http://example.com")
		require.NotNil(t, ns)
		require.Equal(t, "x", ns.Prefix())
	})

	t.Run("xml namespace", func(t *testing.T) {
		doc := CreateDocument()
		e, err := doc.CreateElement("root")
		require.NoError(t, err)

		ns := LookupNSByHref(e, XMLNamespace)
		require.NotNil(t, ns)
		require.Equal(t, "xml", ns.Prefix())
	})

	t.Run("not found", func(t *testing.T) {
		doc := CreateDocument()
		e, err := doc.CreateElement("root")
		require.NoError(t, err)

		ns := LookupNSByHref(e, "http://not.found.com")
		require.Nil(t, ns)
	})
}

func TestLookupNSByPrefix(t *testing.T) {
	doc := CreateDocument()
	e, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, e.SetNamespace("x", "http://example.com"))

	ns := LookupNSByPrefix(e, "x")
	require.NotNil(t, ns)
	require.Equal(t, "http://example.com", ns.URI())

	ns = LookupNSByPrefix(e, "xml")
	require.NotNil(t, ns)
	require.Equal(t, XMLNamespace, ns.URI())

	ns = LookupNSByPrefix(e, "missing")
	require.Nil(t, ns)
}

func TestNodeGetBase(t *testing.T) {
	t.Run("no base", func(t *testing.T) {
		doc := CreateDocument()
		e, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(e))

		base := NodeGetBase(doc, e)
		require.Equal(t, "", base)
	})

	t.Run("direct xml:base", func(t *testing.T) {
		doc := CreateDocument()
		e, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(e))

		xmlNS := NewNamespace("xml", XMLNamespace)
		require.NoError(t, e.SetAttributeNS("base", "http://example.com/", xmlNS))

		base := NodeGetBase(doc, e)
		require.Equal(t, "http://example.com/", base)
	})

	t.Run("inherited xml:base", func(t *testing.T) {
		doc := CreateDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(parent))

		xmlNS := NewNamespace("xml", XMLNamespace)
		require.NoError(t, parent.SetAttributeNS("base", "http://example.com/dir/", xmlNS))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))

		base := NodeGetBase(doc, child)
		require.Equal(t, "http://example.com/dir/", base)
	})

	t.Run("relative resolution", func(t *testing.T) {
		doc := CreateDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(parent))

		xmlNS := NewNamespace("xml", XMLNamespace)
		require.NoError(t, parent.SetAttributeNS("base", "http://example.com/a/b/", xmlNS))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))
		require.NoError(t, child.SetAttributeNS("base", "c/d/", xmlNS))

		base := NodeGetBase(doc, child)
		require.Equal(t, "http://example.com/a/b/c/d/", base)
	})
}

func TestDocumentURL(t *testing.T) {
	t.Run("set and get URL", func(t *testing.T) {
		doc := CreateDocument()
		require.Equal(t, "", doc.URL())

		doc.SetURL("http://example.com/doc.xml")
		require.Equal(t, "http://example.com/doc.xml", doc.URL())
	})

	t.Run("URL used as base in NodeGetBase", func(t *testing.T) {
		doc := CreateDocument()
		doc.SetURL("http://example.com/dir/doc.xml")

		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		base := NodeGetBase(doc, root)
		require.Equal(t, "http://example.com/dir/doc.xml", base)
	})

	t.Run("URL with relative xml:base", func(t *testing.T) {
		doc := CreateDocument()
		doc.SetURL("http://example.com/dir/doc.xml")

		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		xmlNS := NewNamespace("xml", XMLNamespace)
		require.NoError(t, root.SetAttributeNS("base", "sub/", xmlNS))

		base := NodeGetBase(doc, root)
		require.Equal(t, "http://example.com/dir/sub/", base)
	})

	t.Run("URL set during parsing", func(t *testing.T) {
		const input = `<?xml version="1.0"?><root/>`
		p := NewParser()
		p.SetBaseURI("/some/path/doc.xml")
		doc, err := p.Parse([]byte(input))
		require.NoError(t, err)
		require.Equal(t, "/some/path/doc.xml", doc.URL())
	})
}

func TestCopyNode(t *testing.T) {
	t.Run("element with children and attrs", func(t *testing.T) {
		src := CreateDocument()
		root, err := src.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, root.SetAttribute("id", "1"))
		require.NoError(t, src.AddChild(root))

		child, err := src.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(child))
		require.NoError(t, child.AddContent([]byte("hello")))

		dst := CreateDocument()
		copied, err := CopyNode(root, dst)
		require.NoError(t, err)

		elem := copied.(*Element)
		require.Equal(t, "root", elem.LocalName())
		val, ok := elem.GetAttribute("id")
		require.True(t, ok)
		require.Equal(t, "1", val)
		require.NotNil(t, elem.FirstChild())
		require.Equal(t, "child", elem.FirstChild().Name())
		require.Equal(t, "hello", string(elem.FirstChild().Content()))
	})

	t.Run("text node", func(t *testing.T) {
		doc := CreateDocument()
		txt, err := doc.CreateText([]byte("hello"))
		require.NoError(t, err)

		dst := CreateDocument()
		copied, err := CopyNode(txt, dst)
		require.NoError(t, err)
		require.Equal(t, TextNode, copied.Type())
		require.Equal(t, "hello", string(copied.Content()))
	})

	t.Run("comment node", func(t *testing.T) {
		doc := CreateDocument()
		c, err := doc.CreateComment([]byte("a comment"))
		require.NoError(t, err)

		dst := CreateDocument()
		copied, err := CopyNode(c, dst)
		require.NoError(t, err)
		require.Equal(t, CommentNode, copied.Type())
		require.Equal(t, "a comment", string(copied.Content()))
	})

	t.Run("CDATA node", func(t *testing.T) {
		doc := CreateDocument()
		cd, err := doc.CreateCDATASection([]byte("cdata content"))
		require.NoError(t, err)

		dst := CreateDocument()
		copied, err := CopyNode(cd, dst)
		require.NoError(t, err)
		require.Equal(t, CDATASectionNode, copied.Type())
		require.Equal(t, "cdata content", string(copied.Content()))
	})

	t.Run("PI node", func(t *testing.T) {
		doc := CreateDocument()
		pi, err := doc.CreatePI("target", "data")
		require.NoError(t, err)

		dst := CreateDocument()
		copied, err := CopyNode(pi, dst)
		require.NoError(t, err)
		require.Equal(t, ProcessingInstructionNode, copied.Type())
	})

	t.Run("element with namespaces", func(t *testing.T) {
		src := CreateDocument()
		root, err := src.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, root.SetNamespace("x", "http://example.com"))
		require.NoError(t, root.SetNamespace("x", "http://example.com", true))
		require.NoError(t, src.AddChild(root))

		dst := CreateDocument()
		copied, err := CopyNode(root, dst)
		require.NoError(t, err)

		elem := copied.(*Element)
		require.Equal(t, "http://example.com", elem.URI())
	})
}

func TestCopyDoc(t *testing.T) {
	t.Run("document with children", func(t *testing.T) {
		src := CreateDocument()
		root, err := src.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, src.AddChild(root))
		require.NoError(t, root.AddContent([]byte("hello")))

		dst, err := CopyDoc(src)
		require.NoError(t, err)
		require.NotNil(t, dst)
		require.Equal(t, src.Version(), dst.Version())

		dstRoot := dst.DocumentElement()
		require.NotNil(t, dstRoot)
		require.Equal(t, "root", dstRoot.LocalName())
		require.Equal(t, "hello", string(dstRoot.Content()))
	})

	t.Run("document with DTD", func(t *testing.T) {
		src := CreateDocument()
		_, err := src.CreateInternalSubset("root", "", "root.dtd")
		require.NoError(t, err)

		root, err := src.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, src.AddChild(root))

		dst, err := CopyDoc(src)
		require.NoError(t, err)
		require.NotNil(t, dst.IntSubset())
		require.Equal(t, "root", dst.IntSubset().Name())
	})

	t.Run("nil document", func(t *testing.T) {
		_, err := CopyDoc(nil)
		require.Error(t, err)
	})
}

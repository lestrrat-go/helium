package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func mustCreateElement(t *testing.T, doc *helium.Document, name string) *helium.Element {
	t.Helper()
	e, err := doc.CreateElement(name)
	require.NoError(t, err)
	return e
}

func mustCreateText(t *testing.T, doc *helium.Document, text []byte) *helium.Text {
	t.Helper()
	n, err := doc.CreateText(text)
	require.NoError(t, err)
	return n
}

func TestElementTree(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e1 := mustCreateElement(t, doc, "root")
	e2 := mustCreateElement(t, doc, "e2")
	e3 := mustCreateElement(t, doc, "e3")
	e4 := mustCreateElement(t, doc, "e4")
	require.NoError(t, e2.SetAttribute("id", "e2"))
	require.NoError(t, e3.SetAttribute("id", "e3"))
	require.NoError(t, e4.SetAttribute("id", "e4"))

	require.NoError(t, e1.AddChild(e2), "e1.AddChild(e2) succeeds")
	require.NoError(t, e1.AddChild(e3), "e1.AddChild(e3) succeeds")
	require.NoError(t, e1.AddChild(e4), "e1.AddChild(e4) succeeds")

	require.Equal(t, e2, e1.FirstChild(), "e1.FirstChild is e2")
	require.Equal(t, e4, e1.LastChild(), "e1.LastChild is e4")

	require.Equal(t, e3, e2.NextSibling(), "e2.NextSibling is e3")
	require.Equal(t, e4, e3.NextSibling(), "e3.NextSibling is e4")
	require.Equal(t, nil, e4.NextSibling(), "e4.NextSibling is nil")

	require.Equal(t, e3, e4.PrevSibling(), "e4.PrevSibling is e3")
	require.Equal(t, e2, e3.PrevSibling(), "e3.PrevSibling is e2")
	require.Equal(t, nil, e2.PrevSibling(), "e2.PrevSibling is nil")

	require.NoError(t, e2.AppendText([]byte("e2")), "e2.AppendText succeeds")
	require.Equal(t, []byte("e2"), e2.Content(), "e2.Content matches")

	for _, e := range []helium.Node{e2, e3, e4} {
		require.Equal(t, e1, e.Parent(), "%s.Parent is e1", e.Name())
	}

	str, err := e1.XMLString()
	require.NoError(t, err, "e1.XMLString succeeds")
	require.Equal(t, `<root><e2 id="e2">e2</e2><e3 id="e3"/><e4 id="e4"/></root>`, str, "e1.XMLString produces expected result")
}

func TestElementContent(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e := mustCreateElement(t, doc, "root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		require.NoError(t, e.AppendText(chunk), "AppendText succeeds")
	}

	require.IsType(t, &helium.Text{}, e.LastChild(), "LastChild is a Text node")

	require.Equal(t, []byte("Hello World!"), e.Content())

	e = mustCreateElement(t, doc, "root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		require.NoError(t, e.AddChild(mustCreateText(t, doc, chunk)), "AddChild succeeds")
	}

	require.IsType(t, &helium.Text{}, e.LastChild(), "LastChild is a Text node")

	require.Equal(t, []byte("Hello World!"), e.Content())
}

func TestGetAttribute(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, e.SetAttribute("id", "123"))
	require.NoError(t, e.SetAttribute("class", "main"))

	val, ok := e.GetAttribute("id")
	require.True(t, ok)
	require.Equal(t, "123", val)

	val, ok = e.GetAttribute("class")
	require.True(t, ok)
	require.Equal(t, "main", val)

	_, ok = e.GetAttribute("missing")
	require.False(t, ok)
}

func TestHasAttribute(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, e.SetAttribute("id", "123"))

	require.True(t, e.HasAttribute("id"))
	require.False(t, e.HasAttribute("missing"))
}

func TestGetAttributeNS(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e, err := doc.CreateElement("root")
	require.NoError(t, err)

	ns := helium.NewNamespace("x", "http://example.com")
	require.NoError(t, e.SetAttributeNS("attr", "val", ns))

	val, ok := e.GetAttributeNS("attr", "http://example.com")
	require.True(t, ok)
	require.Equal(t, "val", val)

	_, ok = e.GetAttributeNS("attr", "http://other.com")
	require.False(t, ok)

	_, ok = e.GetAttributeNS("missing", "http://example.com")
	require.False(t, ok)
}

func TestGetAttributeNodeNS(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e, err := doc.CreateElement("root")
	require.NoError(t, err)

	ns := helium.NewNamespace("x", "http://example.com")
	require.NoError(t, e.SetAttributeNS("attr", "val", ns))

	attr := e.GetAttributeNodeNS("attr", "http://example.com")
	require.NotNil(t, attr)
	require.Equal(t, "attr", attr.LocalName())
	require.Equal(t, "val", attr.Value())
	require.Equal(t, "http://example.com", attr.URI())

	attr = e.GetAttributeNodeNS("attr", "http://other.com")
	require.Nil(t, attr)

	attr = e.GetAttributeNodeNS("missing", "http://example.com")
	require.Nil(t, attr)
}

func TestRemoveAttribute(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, e.SetAttribute("a", "1"))
	require.NoError(t, e.SetAttribute("b", "2"))
	require.NoError(t, e.SetAttribute("c", "3"))

	// Remove middle
	ok := e.RemoveAttribute("b")
	require.True(t, ok)
	require.False(t, e.HasAttribute("b"))
	require.True(t, e.HasAttribute("a"))
	require.True(t, e.HasAttribute("c"))

	// Remove first
	ok = e.RemoveAttribute("a")
	require.True(t, ok)
	require.False(t, e.HasAttribute("a"))
	require.True(t, e.HasAttribute("c"))

	// Remove last (only remaining)
	ok = e.RemoveAttribute("c")
	require.True(t, ok)
	require.Equal(t, 0, len(e.Attributes()))

	// Remove non-existent
	ok = e.RemoveAttribute("missing")
	require.False(t, ok)
}

func TestRemoveAttributeNS(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e, err := doc.CreateElement("root")
	require.NoError(t, err)

	ns := helium.NewNamespace("x", "http://example.com")
	require.NoError(t, e.SetAttributeNS("attr", "val", ns))

	ok := e.RemoveAttributeNS("attr", "http://example.com")
	require.True(t, ok)
	require.Equal(t, 0, len(e.Attributes()))

	ok = e.RemoveAttributeNS("attr", "http://example.com")
	require.False(t, ok)
}

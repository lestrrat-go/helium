package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func mustCreateElement(t *testing.T, doc *helium.Document, name string) *helium.Element {
	t.Helper()
	e := doc.CreateElement(name)
	return e
}

func mustCreateText(t *testing.T, doc *helium.Document, text []byte) *helium.Text {
	t.Helper()
	n := doc.CreateText(text)
	return n
}

func TestElementTree(t *testing.T) {
	doc := helium.NewDefaultDocument()
	e1 := mustCreateElement(t, doc, "root")
	e2 := mustCreateElement(t, doc, "e2")
	e3 := mustCreateElement(t, doc, "e3")
	e4 := mustCreateElement(t, doc, "e4")
	_, err := e2.SetAttribute("id", "e2")
	require.NoError(t, err)
	_, err = e3.SetAttribute("id", "e3")
	require.NoError(t, err)
	_, err = e4.SetAttribute("id", "e4")
	require.NoError(t, err)

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

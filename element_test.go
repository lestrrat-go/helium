package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElementTree(t *testing.T) {
	e1 := newElement("root")
	e2 := newElement("e2")
	e3 := newElement("e3")
	e4 := newElement("e4")
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

	require.NoError(t, e2.AddContent([]byte("e2")), "e2.AddContent succeeds")
	require.Equal(t, []byte("e2"), e2.Content(), "e2.Content matches")

	for _, e := range []Node{e2, e3, e4} {
		require.Equal(t, e1, e.Parent(), "%s.Parent is e1", e.Name())
	}

	str, err := e1.XMLString()
	require.NoError(t, err, "e1.XMLString succeeds")
	require.Equal(t, `<root><e2 id="e2">e2</e2><e3 id="e3"/><e4 id="e4"/></root>`, str, "e1.XMLString produces expected result")
}

func TestElementContent(t *testing.T) {
	e := newElement("root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		require.NoError(t, e.AddContent(chunk), "AddContent succeeds")
	}

	require.IsType(t, newText(nil), e.LastChild(), "LastChild is a Text node")

	require.Equal(t, []byte("Hello World!"), e.Content())

	e = newElement("root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		require.NoError(t, e.AddChild(newText(chunk)), "AddChild succeeds")
	}

	require.IsType(t, newText(nil), e.LastChild(), "LastChild is a Text node")

	require.Equal(t, []byte("Hello World!"), e.Content())

}
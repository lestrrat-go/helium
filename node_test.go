package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestWalkSeesSiblingReplacementDuringTraversal(t *testing.T) {
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
}

func TestWalkSkipsSiblingRemovedDuringTraversal(t *testing.T) {
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
}

func TestTextAppendText(t *testing.T) {
	doc := helium.NewDefaultDocument()
	n := doc.CreateText([]byte("Hello "))
	require.NoError(t, n.AppendText([]byte("World!")), "AppendText succeeds")
	require.Equal(t, []byte("Hello World!"), n.Content(), "Content matches")
}

func TestTextAddChild(t *testing.T) {
	doc := helium.NewDefaultDocument()
	n1 := doc.CreateText([]byte("Hello "))
	n2 := doc.CreateText([]byte("World!"))

	require.NoError(t, n1.AddChild(n2), "AddChild succeeds")
	require.Equal(t, []byte("Hello World!"), n1.Content(), "Content matches")
}

func TestTextAddChildInvalidNode(t *testing.T) {
	doc := helium.NewDefaultDocument()
	n1 := doc.CreateText([]byte("Hello "))
	n2 := &helium.ProcessingInstruction{}

	require.Equal(t, helium.ErrInvalidOperation, n1.AddChild(n2), "AddChild fails")
	require.Equal(t, []byte("Hello "), n1.Content(), "Content matches")
}

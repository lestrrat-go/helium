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

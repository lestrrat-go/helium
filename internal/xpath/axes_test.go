package xpath_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/stretchr/testify/require"
)

func deepChainDepth(t *testing.T) int {
	t.Helper()

	if testing.Short() {
		return 512
	}
	return 5000
}

func buildDeepChain(t *testing.T, depth int) (*helium.Document, *helium.Element, *helium.Element) {
	t.Helper()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	parent := root
	for range depth {
		child, childErr := doc.CreateElement("level")
		require.NoError(t, childErr)
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	return doc, root, parent
}

func TestTraverseAxisDescendant_DeepChain(t *testing.T) {
	depth := deepChainDepth(t)

	_, root, leaf := buildDeepChain(t, depth)

	nodes, err := ixpath.TraverseAxis(ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Len(t, nodes, depth)
	require.Equal(t, helium.Node(leaf), nodes[len(nodes)-1])
}

func TestTraverseAxisPreceding_DeepChain(t *testing.T) {
	depth := deepChainDepth(t)

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	left, err := doc.CreateElement("left")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(left))

	parent := left
	var leaf helium.Node = left
	for range depth {
		child, childErr := doc.CreateElement("level")
		require.NoError(t, childErr)
		require.NoError(t, parent.AddChild(child))
		parent = child
		leaf = child
	}

	right, err := doc.CreateElement("right")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(right))

	nodes, err := ixpath.TraverseAxis(ixpath.AxisPreceding, right, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Len(t, nodes, depth+1)
	require.Equal(t, leaf, nodes[0])
	require.Equal(t, helium.Node(left), nodes[len(nodes)-1])
}

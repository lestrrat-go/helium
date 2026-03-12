package xpath

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestStringValue_DepthGuardReturnsPartialString(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	prefix, err := doc.CreateText([]byte("prefix"))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(prefix))

	parent := root
	for range maxCollectTextDescendantsDepth {
		child, childErr := doc.CreateElement("level")
		require.NoError(t, childErr)
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	leaf, err := doc.CreateText([]byte("leaf"))
	require.NoError(t, err)
	require.NoError(t, parent.AddChild(leaf))

	require.Equal(t, "prefix", StringValue(root))
}

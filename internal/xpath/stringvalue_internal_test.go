package xpath

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestStringValue_DepthGuardReturnsPartialString(t *testing.T) {
	oldMaxDepth := maxCollectTextDescendantsDepth
	maxCollectTextDescendantsDepth = 2
	t.Cleanup(func() {
		maxCollectTextDescendantsDepth = oldMaxDepth
	})

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	prefix, err := doc.CreateText([]byte("prefix"))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(prefix))

	level1, err := doc.CreateElement("level1")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(level1))

	level2, err := doc.CreateElement("level2")
	require.NoError(t, err)
	require.NoError(t, level1.AddChild(level2))

	leaf, err := doc.CreateText([]byte("leaf"))
	require.NoError(t, err)
	require.NoError(t, level2.AddChild(leaf))

	require.Equal(t, "prefix", StringValue(root))
}

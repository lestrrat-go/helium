package xpath

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestStringValue_DeepTreeDoesNotTruncate(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	prefix := doc.CreateText([]byte("prefix"))
	require.NoError(t, root.AddChild(prefix))

	parent := root
	for range 4096 {
		child := doc.CreateElement("level")
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	leaf := doc.CreateText([]byte("leaf"))
	require.NoError(t, parent.AddChild(leaf))

	require.Equal(t, "prefixleaf", StringValue(root))
}

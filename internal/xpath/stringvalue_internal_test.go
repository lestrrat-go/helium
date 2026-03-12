package xpath

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestStringValue_DepthGuardReturnsPartialString(t *testing.T) {
	const shallowMaxDepth int64 = 8

	// This test mutates package-global MaxStringValueDepth, so it must not run in parallel.
	oldMaxDepth := MaxStringValueDepth.Load()
	MaxStringValueDepth.Store(shallowMaxDepth)
	t.Cleanup(func() {
		MaxStringValueDepth.Store(oldMaxDepth)
	})

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	prefix, err := doc.CreateText([]byte("prefix"))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(prefix))

	parent := root
	for range maxStringValueDepth() {
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

func TestStringValue_DepthGuardFallsBackToDefaultOnInvalidValue(t *testing.T) {
	// This test mutates package-global MaxStringValueDepth, so it must not run in parallel.
	oldMaxDepth := MaxStringValueDepth.Load()
	MaxStringValueDepth.Store(0)
	t.Cleanup(func() {
		MaxStringValueDepth.Store(oldMaxDepth)
	})

	require.Equal(t, int(defaultMaxStringValueDepth), maxStringValueDepth())
}

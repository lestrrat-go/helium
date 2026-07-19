package xpath3_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// A DocOrderCache caches document-order positions when first built. Mutating
// the underlying document leaves those positions stale because BuildFrom is a
// no-op once a root is indexed. Reset clears the cache so the same value can be
// reused after a mutation.
func TestDocOrderCache_Reset_AfterMutation(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	a, err := doc.CreateElement("a")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(a))
	b, err := doc.CreateElement("b")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(b))

	cache := xpath3.NewDocOrderCache()
	cache.BuildFrom(doc)

	// Initial document order: a precedes b.
	require.True(t, cache.Less(a, b), "a should precede b before mutation")

	// Mutate the document: move a after b, so the true order is now b, a.
	require.NoError(t, root.AddChild(a))

	// Without Reset, BuildFrom is a no-op (root already indexed) and the cache
	// still reports the stale pre-mutation order.
	cache.BuildFrom(doc)
	require.True(t, cache.Less(a, b), "stale cache still reports pre-mutation order")

	// After Reset, the cache recomputes order from the current tree.
	cache.Reset()
	cache.BuildFrom(doc)
	require.True(t, cache.Less(b, a), "after Reset b should precede a (current order)")
	require.False(t, cache.Less(a, b), "after Reset a no longer precedes b")
}

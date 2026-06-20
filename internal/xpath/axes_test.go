package xpath_test

import (
	"context"
	"testing"
	"time"

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
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	parent := root
	// go.mod requires Go 1.25, so integer range is part of the supported toolchain.
	for range depth {
		child := doc.CreateElement("level")
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	return doc, root, parent
}

func TestTraverseAxisDescendant_DeepChain(t *testing.T) {
	depth := deepChainDepth(t)

	_, root, leaf := buildDeepChain(t, depth)

	nodes, err := ixpath.TraverseAxis(t.Context(), ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Len(t, nodes, depth)
	require.Equal(t, helium.Node(leaf), nodes[len(nodes)-1])
}

func TestTraverseAxisPreceding_DeepChain(t *testing.T) {
	depth := deepChainDepth(t)

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	left := doc.CreateElement("left")
	require.NoError(t, root.AddChild(left))

	parent := left
	var leaf helium.Node = left
	// go.mod requires Go 1.25, so integer range is part of the supported toolchain.
	for range depth {
		child := doc.CreateElement("level")
		require.NoError(t, parent.AddChild(child))
		parent = child
		leaf = child
	}

	right := doc.CreateElement("right")
	require.NoError(t, root.AddChild(right))

	nodes, err := ixpath.TraverseAxis(t.Context(), ixpath.AxisPreceding, right, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Len(t, nodes, depth+1)
	require.Equal(t, leaf, nodes[0])
	require.Equal(t, helium.Node(left), nodes[len(nodes)-1])
}

// TestTraverseAxisDescendant_ContextCancelled verifies that a context cancelled
// before traversal aborts the descendant walk promptly with context.Canceled
// instead of walking the whole subtree.
func TestTraverseAxisDescendant_ContextCancelled(t *testing.T) {
	// A wide tree so that, absent a context check, traversal would visit a
	// large number of nodes before returning.
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	parent := root
	for range 5000 {
		child := doc.CreateElement("level")
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
}

// cancelAfterNContext reports a cancelled error from Err() only after Err has
// been consulted cancelAfter times, simulating a context that is cancelled
// AFTER traversal has begun (i.e. partway through the walk). It implements
// context.Context directly (no embedding) to satisfy the containedctx linter.
type cancelAfterNContext struct {
	cancelAfter int
	calls       int
}

func (c *cancelAfterNContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterNContext) Done() <-chan struct{}       { return nil }
func (c *cancelAfterNContext) Value(any) any               { return nil }

func (c *cancelAfterNContext) Err() error {
	c.calls++
	if c.calls > c.cancelAfter {
		return context.Canceled
	}
	return nil
}

// TestTraverseAxisDescendant_ContextCancelledMidWalk verifies that a context
// cancelled AFTER the descendant traversal has begun aborts the walk promptly
// rather than completing the full walk over a deep tree. Without an in-loop
// ctx.Err() check the walk would visit all depth nodes regardless.
func TestTraverseAxisDescendant_ContextCancelledMidWalk(t *testing.T) {
	const depth = 20000
	const cancelAfter = 10

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	parent := root
	for range depth {
		child := doc.CreateElement("level")
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
	// The walk must have stopped near the cancellation point, not after walking
	// the entire subtree.
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"traversal should stop on the first cancelled Err() observation")
}

package helium_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func newCycleTestDocument() *helium.Document {
	return helium.NewDocument("1.0", "utf-8", helium.StandaloneExplicitNo)
}

// TestAddChildDeepChainGrowth builds a several-thousand-deep chain through
// AddChild, exercising the same insertion pattern that made wouldCreateCycle's
// ancestor walk quadratic, and asserts the resulting chain's shape rather than
// its timing: every AddChild in the chain must still succeed and land in the
// correct position. Elements carry two attributes each, so each operand IS
// claimed when it is inserted and the guard settles it through the claim search
// rather than the bare claim-count exit the attribute-free case takes.
func TestAddChildDeepChainGrowth(t *testing.T) {
	const depth = 4000

	doc := newCycleTestDocument()
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))

	cur := helium.Node(root)
	nodes := make([]*helium.Element, 0, depth)
	for i := range depth {
		e, err := doc.CreateElement(fmt.Sprintf("e%d", i))
		require.NoError(t, err)
		require.NoError(t, e.SetAttribute("a0", "v0"))
		require.NoError(t, e.SetAttribute("a1", "v1"))
		mut, ok := cur.(helium.MutableNode)
		require.True(t, ok)
		require.NoError(t, mut.AddChild(e))
		nodes = append(nodes, e)
		cur = e
	}

	// Walk back down verifying parent/child linkage and that no node was
	// mislinked or skipped.
	walker := helium.Node(root)
	for i, e := range nodes {
		child := walker.FirstChild()
		require.NotNil(t, child, "missing child at depth %d", i)
		require.Same(t, helium.Node(e), child)
		require.Same(t, walker, child.Parent(), "parent mismatch at depth %d", i)
		require.Nil(t, child.NextSibling(), "unexpected sibling at depth %d", i)
		walker = child
	}

	// The deepest node must be a true leaf.
	require.Nil(t, walker.FirstChild())
}

// TestUnsafeSetParentClaimantIsRejected pins that the insertion cycle guard
// rejects a parent-pointer loop whichever way the claimant was written. The
// guard settles a childless operand from the number of nodes naming it as their
// parent, and UnsafeSetParent maintains that count exactly as the safe paths do,
// so a link it wrote into no slot at all still keeps the guard on its ancestor
// walk. Both shapes must therefore fail with ErrCyclicNode and leave the tree
// unlinked.
func TestUnsafeSetParentClaimantIsRejected(t *testing.T) {
	t.Run("UnsafeSetParent link is rejected", func(t *testing.T) {
		doc := newCycleTestDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		cur, err := doc.CreateElement("cur")
		require.NoError(t, err)

		// cur becomes parent's parent without being linked into any slot: no
		// child list, no properties chain, no DTD subset. Only the claim count
		// records it.
		helium.UnsafeSetParent(parent, cur)

		err = parent.AddChild(cur)
		require.True(t, errors.Is(err, helium.ErrCyclicNode), "expected ErrCyclicNode, got %v", err)
		require.Nil(t, parent.FirstChild(), "the rejected insertion must not link cur")
	})

	t.Run("same shape through the safe API is rejected", func(t *testing.T) {
		doc := newCycleTestDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		cur, err := doc.CreateElement("cur")
		require.NoError(t, err)

		require.NoError(t, cur.AddChild(parent))

		err = parent.AddChild(cur)
		require.True(t, errors.Is(err, helium.ErrCyclicNode), "expected ErrCyclicNode, got %v", err)
	})
}

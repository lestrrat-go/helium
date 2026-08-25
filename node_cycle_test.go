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
// correct position. Elements carry two attributes each so the properties-chain
// fast path (propertiesReach) is exercised on every insertion, not just the
// zero-attribute case.
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

// TestUnsafeSetParentIsOutsideCycleGuardContract pins where the insertion cycle
// guard's contract ends. The guard resolves a childless operand from that
// operand's own claimant sources — its child list, an Element's attributes, a
// Document's DTD subsets — rather than by walking parent pointers, so a parent
// link written by UnsafeSetParent, which links nothing into any of them, is
// invisible to it and the insertion that closes the loop is accepted. The same
// shape built through the safe API is rejected. Both halves are asserted here so
// the boundary is recorded in the suite, not only in the doc comments on
// wouldCreateCycle and UnsafeSetParent.
func TestUnsafeSetParentIsOutsideCycleGuardContract(t *testing.T) {
	t.Run("UnsafeSetParent link is accepted", func(t *testing.T) {
		doc := newCycleTestDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		cur, err := doc.CreateElement("cur")
		require.NoError(t, err)

		// cur becomes parent's parent without being linked as anyone's parent
		// in a child list, which is precisely what the guard does not track.
		helium.UnsafeSetParent(parent, cur)

		require.NoError(t, parent.AddChild(cur), "a tree corrupted through UnsafeSetParent is outside the guard's contract")
		require.Same(t, helium.Node(cur), parent.FirstChild())
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

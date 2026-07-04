package helium

import (
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestChildReachesNoDepthCap verifies the child-pointer reachability search has
// no depth cap: a target reachable only past the old 4096-node cap is still
// found. A capped search would fail OPEN (return false = "not reachable"),
// letting a cycle deeper than the cap slip through the insertion guard.
func TestChildReachesNoDepthCap(t *testing.T) {
	doc := NewDefaultDocument()

	// A single-child chain first -> n -> n -> ... -> deep, deeper than the
	// removed 4096 cap. childReaches must descend the whole chain.
	const depth = 5000
	first := doc.CreateElement("n0")
	prev := first
	for i := 1; i <= depth; i++ {
		cur := doc.CreateElement("n")
		require.NoError(t, prev.AddChild(cur))
		prev = cur
	}

	require.True(t, childReaches(first, prev.baseDocNode()),
		"a target %d levels deep must be found — the search must not cap depth and fail open", depth)

	// A node NOT in the subtree is not reachable.
	outside := doc.CreateElement("outside")
	require.False(t, childReaches(first, outside.baseDocNode()),
		"a node outside the subtree must not be reported reachable")
}

// TestChildReachesTerminatesOnCyclicSiblingList verifies the child-pointer
// reachability search terminates when a node's child list has a cyclic sibling
// pointer. The popped-node visited set alone does not bound the inner sibling
// enumeration, so a self-referential or 2-cycle sibling link would spin forever
// without the per-list sibling guard.
func TestChildReachesTerminatesOnCyclicSiblingList(t *testing.T) {
	t.Run("self-cycle", func(t *testing.T) {
		doc := NewDefaultDocument()
		parent := doc.CreateElement("parent")
		c := doc.CreateElement("c")
		require.NoError(t, parent.AddChild(c))

		// Corrupt the sibling list into a self-cycle: c.next = c.
		c.SetNextSibling(c)

		outside := doc.CreateElement("outside")
		require.False(t, childReaches(parent, outside.baseDocNode()),
			"a target not in the cyclic list must terminate and report false")
		require.True(t, childReaches(parent, c.baseDocNode()),
			"a node in the cyclic list must still be found")
	})

	t.Run("two-cycle", func(t *testing.T) {
		doc := NewDefaultDocument()
		parent := doc.CreateElement("parent")
		c1 := doc.CreateElement("c1")
		c2 := doc.CreateElement("c2")
		require.NoError(t, parent.AddChild(c1))
		require.NoError(t, parent.AddChild(c2))

		// Close a 2-cycle: c1 -> c2 -> c1.
		c2.SetNextSibling(c1)

		outside := doc.CreateElement("outside")
		require.False(t, childReaches(parent, outside.baseDocNode()),
			"a target not in the cyclic list must terminate and report false")
	})
}

// TestWalkRejectsEntityChildCycle verifies Walk returns ErrWalkCycle (rather
// than looping forever) on a child-pointer cycle that the guarded insertion API
// refuses to build: an entity reference's shared Entity child links back to the
// reference (ent.firstChild = ref, ref.firstChild = ent). The cycle is closed
// through the lower-level docnode links, bypassing AddChild.
func TestWalkRejectsEntityChildCycle(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	ref, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.Equal(t, ent, ref.FirstChild(), "reference's child is the shared Entity node")

	// Close the cycle the ancestor-and-child insertion guard would reject.
	ent.firstChild = ref
	ent.lastChild = ref

	err = Walk(ref, NodeWalkerFunc(func(Node) error { return nil }))
	require.ErrorIs(t, err, ErrWalkCycle,
		"Walk must detect the child-pointer cycle and return ErrWalkCycle instead of hanging")
}

// TestContentTerminatesOnChildPointerCycle verifies the aggregating Content()
// terminates on a pure child-pointer cycle NOT routed through a terminating
// Entity: a -> b -> a, built with the low-level link primitives that bypass the
// guarded AddChild. Without the active-path guard the container recursion would
// recurse forever (stack overflow); with it, the back-edge into a (already on
// the active path) is skipped and the sibling text is still aggregated.
func TestContentTerminatesOnChildPointerCycle(t *testing.T) {
	doc := NewDefaultDocument()
	a := doc.CreateElement("a")
	txt := doc.CreateText([]byte("x"))
	b := doc.CreateElement("b")
	require.NoError(t, a.AddChild(txt))
	require.NoError(t, a.AddChild(b))

	// Close a child-pointer cycle a -> b -> a: b's only child is a (parent link
	// included so it is a genuine owned child, invisible to the owned-boundary
	// advance and caught only by the active-path guard).
	setFirstChild(b, a)
	setLastChild(b, a)
	a.SetParent(b)

	require.Equal(t, []byte("x"), a.Content(),
		"Content must terminate on the child-pointer back-edge and still aggregate the text sibling")
}

// TestWalkRejectsSiblingCycle verifies Walk terminates (rather than spinning
// forever) on a sibling cycle LONGER than one node: a parent whose two children
// form a 2-cycle a -> b -> a. The active-path guard alone does not catch this —
// each child is popped off the stack before its next sibling is examined — so
// the per-frame seenChildren set must return ErrWalkCycle.
func TestWalkRejectsSiblingCycle(t *testing.T) {
	doc := NewDefaultDocument()
	parent := doc.CreateElement("parent")
	a := doc.CreateElement("a")
	b := doc.CreateElement("b")
	require.NoError(t, parent.AddChild(a))
	require.NoError(t, parent.AddChild(b))

	// Close a 2-cycle in the sibling list: a -> b -> a.
	b.SetNextSibling(a)

	err := Walk(parent, NodeWalkerFunc(func(Node) error { return nil }))
	require.ErrorIs(t, err, ErrWalkCycle,
		"Walk must detect the sibling cycle and return ErrWalkCycle instead of hanging")
}

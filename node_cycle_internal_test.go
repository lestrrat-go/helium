package helium

import (
	"testing"

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

package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestAddSiblingTailGrowth appends N siblings through the FIRST child, which
// is a non-tail anchor after the first append. This is a correctness test
// (not a timing test): it pins the shape of the resulting child list, which
// is exactly what the O(1) tail-jump in addSibling must still produce.
func TestAddSiblingTailGrowth(t *testing.T) {
	t.Parallel()

	const n = 2000

	doc := helium.NewDefaultDocument()
	parent := mustCreateElement(t, doc, "parent")
	first := mustCreateElement(t, doc, "n0")
	require.NoError(t, parent.AddChild(first))

	appended := make([]*helium.Element, n)
	for i := range n {
		child := mustCreateElement(t, doc, "n")
		require.NoError(t, first.AddSibling(child), "append %d", i)
		appended[i] = child
	}

	require.Equal(t, helium.Node(first), parent.FirstChild())
	require.Equal(t, helium.Node(appended[n-1]), parent.LastChild())

	prev := helium.Node(first)
	for i, child := range appended {
		require.Equal(t, prev, child.PrevSibling(), "prev symmetry at %d", i)
		require.Equal(t, helium.Node(child), prev.NextSibling(), "next symmetry at %d", i)
		require.Equal(t, parent, child.Parent(), "parent at %d", i)
		prev = child
	}
	require.Nil(t, appended[n-1].NextSibling(), "last node must be the true tail")

	count := 0
	for range helium.Children(parent) {
		count++
	}
	require.Equal(t, n+1, count, "child count including the anchor")
}

// TestAddSiblingRepairsStaleLastChild pins the tdn.next == nil guard: when
// parent.lastChild is stale (not the true tail), AddChild's repair path calls
// AddSibling on the stale lastChild, and the guard must reject the O(1) jump
// so the fallback walk finds and repairs the true tail.
func TestAddSiblingRepairsStaleLastChild(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	parent := mustCreateElement(t, doc, "parent")
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")
	c := mustCreateElement(t, doc, "c")
	require.NoError(t, parent.AddChild(a))
	require.NoError(t, parent.AddChild(b))
	require.NoError(t, parent.AddChild(c))
	require.Equal(t, c, parent.LastChild())

	// Corrupt parent.lastChild into staleness: link d after c via raw pointer
	// writes, bypassing AddChild/AddSibling so parent.lastChild is left
	// pointing at c even though d is now the true tail.
	d := mustCreateElement(t, doc, "d")
	helium.UnsafeSetNextSibling(c, d)
	helium.UnsafeSetPrevSibling(d, c)
	helium.UnsafeSetParent(d, parent)

	e := mustCreateElement(t, doc, "e")
	require.NoError(t, parent.AddChild(e))

	require.Equal(t, helium.Node(e), d.NextSibling(), "e must land after the TRUE tail d")
	require.Equal(t, helium.Node(d), e.PrevSibling())
	require.Equal(t, parent, e.Parent())
	require.Equal(t, e, parent.LastChild(), "lastChild must be repaired to the true tail")
}

// TestAddSiblingParentlessChain covers a detached sibling chain with no
// owning parent: there is no lastChild to jump to, so AddSibling through a
// non-tail anchor must fall back to the walk and still land at the true tail.
func TestAddSiblingParentlessChain(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")
	c := mustCreateElement(t, doc, "c")
	require.NoError(t, a.AddSibling(b))
	require.NoError(t, b.AddSibling(c))

	d := mustCreateElement(t, doc, "d")
	require.NoError(t, a.AddSibling(d))

	require.Equal(t, helium.Node(d), c.NextSibling(), "d must append at the end")
	require.Equal(t, helium.Node(c), d.PrevSibling())
	require.Nil(t, d.Parent(), "a detached chain has no owner to assign as parent")
}

// TestAddSiblingAttributeChain covers a property attribute anchor: AddSibling
// must append at the end of the attribute chain and never touch the owning
// element's firstChild/lastChild.
func TestAddSiblingAttributeChain(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	elem := mustCreateElement(t, doc, "elem")
	attr1, err := doc.CreateAttribute("a1", "v1", nil)
	require.NoError(t, err)
	attr2, err := doc.CreateAttribute("a2", "v2", nil)
	require.NoError(t, err)
	require.NoError(t, elem.AddChild(attr1))
	require.NoError(t, elem.AddChild(attr2))

	attr3, err := doc.CreateAttribute("a3", "v3", nil)
	require.NoError(t, err)
	// attr1 is a non-tail anchor: attr2 already follows it in the properties
	// chain.
	require.NoError(t, attr1.AddSibling(attr3))

	require.Equal(t, helium.Node(attr3), attr2.NextSibling(), "attr3 appends after the true attribute tail")
	require.Nil(t, elem.FirstChild(), "the attribute chain must never touch firstChild")
	require.Nil(t, elem.LastChild(), "the attribute chain must never touch lastChild")

	attrs := elem.Attributes()
	require.Len(t, attrs, 3)
}

// TestAddSiblingAttributeInChildList covers an *Attribute that a generic
// Replace placed in the ordinary child list rather than the owning element's
// properties chain. AddSibling on such an attribute must still take the
// generic child-list branch (and its O(1) tail jump), not the property-list
// branch.
func TestAddSiblingAttributeInChildList(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	parent := mustCreateElement(t, doc, "parent")
	placeholder := mustCreateElement(t, doc, "placeholder")
	require.NoError(t, parent.AddChild(placeholder))

	attrX, err := doc.CreateAttribute("x", "vx", nil)
	require.NoError(t, err)
	// A generic Replace swaps a normal child for an attribute, landing attrX
	// in parent's child list instead of parent.properties.
	require.NoError(t, placeholder.Replace(attrX))
	require.Equal(t, helium.Node(attrX), parent.FirstChild())
	require.Equal(t, helium.Node(attrX), parent.LastChild())

	attrY, err := doc.CreateAttribute("y", "vy", nil)
	require.NoError(t, err)
	require.NoError(t, attrX.AddSibling(attrY))

	require.Equal(t, attrY, parent.LastChild(), "AddSibling must take the generic child-list branch")
	require.Equal(t, parent, attrY.Parent())
	require.Empty(t, parent.Attributes(), "the property chain must be untouched")
}

// unlinkedParentClaimTree builds a parent whose child list is [a b] while
// parent.lastChild records a node that list does not contain. The detour node
// claims parent through UnsafeSetParent without being linked into the child
// list, so appending a sibling through it moves lastChild off the chain.
// It returns the parent, its second (and last reachable) child, and the node
// lastChild now records.
func unlinkedParentClaimTree(t *testing.T) (*helium.Element, *helium.Element, *helium.Element) {
	t.Helper()

	doc := helium.NewDefaultDocument()
	parent := mustCreateElement(t, doc, "parent")
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")
	require.NoError(t, parent.AddChild(a))
	require.NoError(t, parent.AddChild(b))

	claimant := mustCreateElement(t, doc, "claimant")
	helium.UnsafeSetParent(claimant, parent)
	recorded := mustCreateElement(t, doc, "recorded")
	require.NoError(t, claimant.AddSibling(recorded))

	require.Equal(t, recorded, parent.LastChild(), "lastChild records a node outside the child list")
	require.Equal(t, helium.Node(a), parent.FirstChild())
	require.Nil(t, b.NextSibling(), "the reachable chain still ends at b")

	return parent, b, recorded
}

// elementNames collects the names of a parent's reachable children.
func elementNames(t *testing.T, parent *helium.Element) []string {
	t.Helper()

	var names []string
	for child := range helium.Children(parent) {
		elem, ok := child.(*helium.Element)
		require.True(t, ok)
		names = append(names, elem.Name())
	}
	return names
}

// TestAddSiblingUnlinkedParentClaim pins the CURRENT, DELIBERATE behavior of
// the O(1) tail jump on a tree that is outside its contract: parent.lastChild
// records a node the parent's child list does not contain, because a raw
// UnsafeSetParent write put a node under parent without linking it in. The
// jump appends after that recorded tail, which is exactly where AddChild and
// UnsafeAppendChild put it on the same tree — all three trust lastChild, and
// the jump's guards are the strictest of the three.
//
// This pins a documented out-of-contract case so the boundary is executable.
// It is not an endorsement of building a tree this shape: on any tree built
// through the safe API alone, every append lands at the end of the reachable
// child list.
func TestAddSiblingUnlinkedParentClaim(t *testing.T) {
	t.Parallel()

	t.Run("AddSibling through a non-tail anchor", func(t *testing.T) {
		t.Parallel()

		parent, b, recorded := unlinkedParentClaimTree(t)
		doc := parent.OwnerDocument()
		anchor, ok := parent.FirstChild().(*helium.Element)
		require.True(t, ok)

		added := mustCreateElement(t, doc, "added")
		require.NoError(t, anchor.AddSibling(added))

		require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the reachable child list is unchanged")
		require.Nil(t, b.NextSibling(), "the reachable chain still ends at b")
		require.Equal(t, added, parent.LastChild(), "lastChild advances to the appended node")
		require.Equal(t, helium.Node(recorded), added.PrevSibling(), "the append lands after the recorded tail")
		require.Equal(t, helium.Node(added), recorded.NextSibling())
		require.Equal(t, parent, added.Parent())
	})

	t.Run("AddChild reaches the same shape", func(t *testing.T) {
		t.Parallel()

		parent, b, recorded := unlinkedParentClaimTree(t)
		doc := parent.OwnerDocument()

		added := mustCreateElement(t, doc, "added")
		require.NoError(t, parent.AddChild(added))

		require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the reachable child list is unchanged")
		require.Nil(t, b.NextSibling())
		require.Equal(t, added, parent.LastChild())
		require.Equal(t, helium.Node(recorded), added.PrevSibling(), "AddChild trusts the same recorded tail")
	})

	t.Run("UnsafeAppendChild reaches the same shape", func(t *testing.T) {
		t.Parallel()

		parent, b, recorded := unlinkedParentClaimTree(t)
		doc := parent.OwnerDocument()

		added := mustCreateElement(t, doc, "added")
		require.NoError(t, helium.UnsafeAppendChild(parent, added))

		require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the reachable child list is unchanged")
		require.Nil(t, b.NextSibling())
		require.Equal(t, added, parent.LastChild())
		require.Equal(t, helium.Node(recorded), added.PrevSibling(), "UnsafeAppendChild trusts the same recorded tail")
	})
}

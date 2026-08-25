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

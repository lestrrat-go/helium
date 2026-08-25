package helium_test

import (
	"os"
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

// offChainParentClaimTree builds a parent whose child list is [a b] while a
// third node claims that parent through UnsafeSetParent without ever being
// linked into the child list. It returns the parent, its last reachable child,
// and the off-chain claimant.
func offChainParentClaimTree(t *testing.T) (*helium.Element, *helium.Element, *helium.Element) {
	t.Helper()

	doc := helium.NewDefaultDocument()
	parent := mustCreateElement(t, doc, "parent")
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")
	require.NoError(t, parent.AddChild(a))
	require.NoError(t, parent.AddChild(b))

	claimant := mustCreateElement(t, doc, "claimant")
	helium.UnsafeSetParent(claimant, parent)
	trailer := mustCreateElement(t, doc, "trailer")
	require.NoError(t, claimant.AddSibling(trailer))

	require.Equal(t, helium.Node(trailer), claimant.NextSibling(), "the append lands after the off-chain anchor")
	require.Equal(t, parent, trailer.Parent())
	require.Equal(t, helium.Node(a), parent.FirstChild())
	require.Equal(t, b, parent.LastChild(), "an off-chain append must not move lastChild off the child list")
	require.Nil(t, b.NextSibling(), "the reachable chain still ends at b")
	requireLastChildOnChain(t, parent)

	return parent, b, claimant
}

// requireLastChildOnChain asserts the invariant addSibling's O(1) tail jump
// rests on: parent.lastChild is the final node of the chain that starts at
// parent.firstChild, never a node outside it.
func requireLastChildOnChain(t *testing.T, parent helium.Node) {
	t.Helper()

	var last helium.Node
	for child := range helium.Children(parent) {
		last = child
	}
	require.Equal(t, last, parent.LastChild(), "lastChild must be the tail of the reachable child chain")
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

// TestAddSiblingOffChainParentClaim covers a node that claims a parent without
// being a member of that parent's child list. Appending a sibling through such
// a node must not move parent.lastChild off the child list, and a later append
// through an on-chain anchor must still land at the end of the REACHABLE child
// list, never after the off-chain node. All three append entry points agree.
func TestAddSiblingOffChainParentClaim(t *testing.T) {
	t.Parallel()

	t.Run("AddSibling through a non-tail anchor", func(t *testing.T) {
		t.Parallel()

		parent, b, claimant := offChainParentClaimTree(t)
		doc := parent.OwnerDocument()
		anchor, ok := parent.FirstChild().(*helium.Element)
		require.True(t, ok)

		added := mustCreateElement(t, doc, "added")
		require.NoError(t, anchor.AddSibling(added))

		require.Equal(t, []string{"a", "b", "added"}, elementNames(t, parent))
		require.Equal(t, helium.Node(b), added.PrevSibling(), "the append lands after the reachable tail")
		require.Equal(t, added, parent.LastChild())
		require.Equal(t, parent, added.Parent())
		require.Nil(t, claimant.PrevSibling(), "the off-chain chain is untouched")
		requireLastChildOnChain(t, parent)
	})

	t.Run("AddChild reaches the same shape", func(t *testing.T) {
		t.Parallel()

		parent, b, _ := offChainParentClaimTree(t)
		doc := parent.OwnerDocument()

		added := mustCreateElement(t, doc, "added")
		require.NoError(t, parent.AddChild(added))

		require.Equal(t, []string{"a", "b", "added"}, elementNames(t, parent))
		require.Equal(t, helium.Node(b), added.PrevSibling())
		require.Equal(t, added, parent.LastChild())
		requireLastChildOnChain(t, parent)
	})

	t.Run("UnsafeAppendChild reaches the same shape", func(t *testing.T) {
		t.Parallel()

		parent, b, _ := offChainParentClaimTree(t)
		doc := parent.OwnerDocument()

		added := mustCreateElement(t, doc, "added")
		require.NoError(t, helium.UnsafeAppendChild(parent, added))

		require.Equal(t, []string{"a", "b", "added"}, elementNames(t, parent))
		require.Equal(t, helium.Node(b), added.PrevSibling())
		require.Equal(t, added, parent.LastChild())
		requireLastChildOnChain(t, parent)
	})
}

// TestAddSiblingCopiedExternalSubsetClaim reaches the same off-chain parent
// claim through public, non-Unsafe calls only. CopyExtSubset installs the
// copied external subset as a *Document's extSubset and gives it that document
// as its parent, but the external subset is not a member of the document's
// child list. Appending a sibling through it must therefore leave the
// document's recorded tail on the child list, so a later append through a
// genuine child still lands at the end of the reachable children.
func TestAddSiblingCopiedExternalSubsetClaim(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := dir + "/ext.dtd"
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`

	src, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	require.NotNil(t, src.ExtSubset())

	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	comment := dst.CreateComment([]byte("lead"))
	root := mustCreateElement(t, dst, "root")
	require.NoError(t, dst.AddChild(comment))
	require.NoError(t, dst.AddChild(root))

	helium.CopyExtSubset(src, dst)
	extSubset := dst.ExtSubset()
	require.NotNil(t, extSubset)
	require.Equal(t, helium.Node(dst), extSubset.Parent(), "the copied external subset claims the document as its parent")

	// The external subset is not in dst's child list, so appending through it
	// must not retarget dst.lastChild at the appended node.
	stray := mustCreateElement(t, dst, "stray")
	require.NoError(t, extSubset.AddSibling(stray))
	require.Equal(t, root, dst.LastChild(), "lastChild stays on the document's child list")
	requireLastChildOnChain(t, dst)

	// A later append through a genuine child must land after root, not after
	// the off-chain stray.
	added := mustCreateElement(t, dst, "added")
	require.NoError(t, comment.AddSibling(added))

	var names []string
	for child := range helium.Children(dst) {
		names = append(names, child.Type().String())
	}
	require.Equal(t, []string{"CommentNode", "ElementNode", "ElementNode"}, names)
	require.Equal(t, helium.Node(root), added.PrevSibling(), "the append lands after the reachable tail")
	require.Equal(t, helium.Node(added), root.NextSibling())
	require.Equal(t, added, dst.LastChild())
	requireLastChildOnChain(t, dst)
}

// TestAddSiblingForgedPrevEdge pins the reciprocity requirement chainMember's
// prev walk rests on. A raw UnsafeSetPrevSibling write can point a node at a
// genuine child of the parent it claims WITHOUT that child pointing back, so
// walking prev alone reaches parent.firstChild and would "prove" a membership
// that does not exist. The anchor here sits on its own two-node chain, so the
// append must join THAT chain and leave the parent's real child list and its
// recorded tail untouched.
func TestAddSiblingForgedPrevEdge(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	parent := mustCreateElement(t, doc, "parent")
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")
	require.NoError(t, parent.AddChild(a))
	require.NoError(t, parent.AddChild(b))

	// x and y form a detached two-node chain of their own.
	x := mustCreateElement(t, doc, "x")
	y := mustCreateElement(t, doc, "y")
	require.NoError(t, x.AddSibling(y))

	// x now claims parent and points its prev edge at a, but a still points
	// forward to b: the edge is one-way, so x is not on the parent's chain.
	helium.UnsafeSetParent(x, parent)
	helium.UnsafeSetPrevSibling(x, a)

	added := mustCreateElement(t, doc, "added")
	require.NoError(t, x.AddSibling(added))

	require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the parent's child list is unchanged")
	require.Equal(t, b, parent.LastChild(), "a forged prev edge must not move lastChild")
	require.Nil(t, b.NextSibling(), "the reachable chain still ends at b")
	requireLastChildOnChain(t, parent)

	require.Equal(t, helium.Node(y), added.PrevSibling(), "the append joins the anchor's own chain")
	require.Equal(t, helium.Node(added), y.NextSibling(), "the anchor's chain keeps its own tail link")
}

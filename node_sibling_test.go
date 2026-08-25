package helium_test

import (
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// nameAdded is the name every appended node in this file is given, so an
// assertion on a child list reads as the position the append landed in.
const nameAdded = "added"

// TestAddSiblingTailGrowth appends N siblings through the FIRST child, which
// is a non-tail anchor after the first append. This is a correctness test
// (not a timing test): it pins the shape of the resulting child list, which is
// exactly what addSibling's O(1) append-point resolution must still produce.
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

// TestAddSiblingRepairsStaleLastChild pins the stale-lastChild repair path:
// when parent.lastChild is not the true tail, AddChild calls AddSibling on that
// stale record, and the append must still find and repair the true tail.
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
// generic child-list branch (and its O(1) append-point resolution), not the
// property-list branch.
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
// linked into the child list. A sibling is then appended through that claimant.
// It returns the parent, its last reachable child, and the trailer that append
// left behind — which the parent now records as its lastChild even though the
// reachable child list still ends at b.
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
	require.Equal(t, trailer, parent.LastChild(), "the append records its own node as the tail, off the child list")
	require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the reachable child list is unchanged")
	require.Nil(t, b.NextSibling(), "the reachable chain still ends at b")

	return parent, b, trailer
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
// being a member of that parent's child list. An append through such a node
// records its own result as the parent's lastChild, so the record leaves the
// child list. The three append entry points then part company, and each one
// keeps the position it has always had: AddSibling walks from its anchor and so
// lands at the end of the REACHABLE child list, while AddChild and
// UnsafeAppendChild both start from the recorded lastChild and so land after the
// off-chain node. The O(1) tail resolution changes none of that — it declines
// for a document with a hand-written link and lets AddSibling walk.
func TestAddSiblingOffChainParentClaim(t *testing.T) {
	t.Parallel()

	t.Run("AddSibling walks to the reachable tail", func(t *testing.T) {
		t.Parallel()

		parent, b, trailer := offChainParentClaimTree(t)
		doc := parent.OwnerDocument()
		anchor, ok := parent.FirstChild().(*helium.Element)
		require.True(t, ok)

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, anchor.AddSibling(added))

		require.Equal(t, []string{"a", "b", nameAdded}, elementNames(t, parent))
		require.Equal(t, helium.Node(b), added.PrevSibling(), "the append lands after the reachable tail")
		require.Equal(t, added, parent.LastChild())
		require.Equal(t, parent, added.Parent())
		require.Nil(t, trailer.NextSibling(), "the off-chain chain is untouched")
	})

	t.Run("AddChild appends after the recorded tail", func(t *testing.T) {
		t.Parallel()

		parent, _, trailer := offChainParentClaimTree(t)
		doc := parent.OwnerDocument()

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, parent.AddChild(added))

		require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the reachable child list is unchanged")
		require.Equal(t, helium.Node(trailer), added.PrevSibling(), "AddChild starts from the recorded tail")
		require.Equal(t, added, parent.LastChild())
	})

	t.Run("UnsafeAppendChild appends after the recorded tail", func(t *testing.T) {
		t.Parallel()

		parent, _, trailer := offChainParentClaimTree(t)
		doc := parent.OwnerDocument()

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, helium.UnsafeAppendChild(parent, added))

		require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the reachable child list is unchanged")
		require.Equal(t, helium.Node(trailer), added.PrevSibling(), "UnsafeAppendChild starts from the recorded tail")
		require.Equal(t, added, parent.LastChild())
	})
}

// TestAddSiblingCopiedExternalSubsetClaim reaches the same off-chain parent
// claim through public, non-Unsafe calls only. CopyExtSubset installs the
// copied external subset as a *Document's extSubset and gives it that document
// as its parent, but the external subset is not a member of the document's
// child list. An append through it therefore moves the document's recorded tail
// off that child list, and a later append through a genuine child must still
// walk to the end of the reachable children — which is what it does, because
// CopyExtSubset records its off-chain parent claim and the O(1) tail resolution
// declines for the destination document from then on.
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
	// records a tail that is not on that list.
	stray := mustCreateElement(t, dst, "stray")
	require.NoError(t, extSubset.AddSibling(stray))
	require.Equal(t, stray, dst.LastChild(), "the append records its own node as the tail")
	require.Nil(t, root.NextSibling(), "the reachable child list still ends at root")

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
	require.Nil(t, stray.NextSibling(), "the off-chain chain is untouched")
}

// TestAddSiblingCorruptShapesMatchWalk drives the corrupt tree shapes that no
// guarded path can build — a hand-forged next edge, a hand-forged prev edge, a
// hand-written parent claim, and a recorded tail cut out of the chain behind it
// — through the public AddSibling API, and pins the exact tree each append
// leaves behind. Every expectation here is the tree the plain sibling walk
// produces, so the whole test also passes unchanged against the walk-only
// implementation: that is what makes it a differential check rather than a
// restatement of the current code. The O(1) tail resolution declines for a
// document whose links were written by hand, which is why the two agree.
func TestAddSiblingCorruptShapesMatchWalk(t *testing.T) {
	t.Parallel()

	t.Run("recorded tail cut out of the chain", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDefaultDocument()
		parent := mustCreateElement(t, doc, "parent")
		a := mustCreateElement(t, doc, "a")
		b := mustCreateElement(t, doc, "b")
		c := mustCreateElement(t, doc, "c")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))
		require.NoError(t, parent.AddChild(c))

		// Cut c out of the chain from the front while leaving it recorded as
		// parent.lastChild, still claiming parent and still ending its own chain.
		// Every pointer the O(1) resolution can read still looks healthy.
		helium.UnsafeSetNextSibling(b, nil)
		require.Equal(t, c, parent.LastChild())

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, a.AddSibling(added))

		require.Equal(t, []string{"a", "b", nameAdded}, elementNames(t, parent), "the append lands at the end of the reachable chain")
		require.Equal(t, helium.Node(b), added.PrevSibling())
		require.Equal(t, added, parent.LastChild())
		require.Nil(t, c.NextSibling(), "the excised node keeps its own links")
		require.Equal(t, helium.Node(b), c.PrevSibling())
	})

	t.Run("recorded tail claiming another parent", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDefaultDocument()
		parent := mustCreateElement(t, doc, "parent")
		other := mustCreateElement(t, doc, "other")
		a := mustCreateElement(t, doc, "a")
		b := mustCreateElement(t, doc, "b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		// b still ends parent's chain, but now claims a different owner.
		helium.UnsafeSetParent(b, other)

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, a.AddSibling(added))

		require.Equal(t, helium.Node(added), b.NextSibling(), "the walk still ends at b and links there")
		require.Equal(t, helium.Node(b), added.PrevSibling())
		require.Equal(t, other, added.Parent(), "the appended node takes the parent the walk ended under")
		require.Equal(t, added, other.LastChild(), "and that parent is the one whose tail is recorded")
		require.Equal(t, b, parent.LastChild(), "parent's own record is left where it was")
	})

	t.Run("forged prev edge into another chain", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDefaultDocument()
		parent := mustCreateElement(t, doc, "parent")
		a := mustCreateElement(t, doc, "a")
		b := mustCreateElement(t, doc, "b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		x := mustCreateElement(t, doc, "x")
		y := mustCreateElement(t, doc, "y")
		require.NoError(t, x.AddSibling(y))
		// x claims parent and points back at a, but a still points forward at b.
		helium.UnsafeSetParent(x, parent)
		helium.UnsafeSetPrevSibling(x, a)

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, x.AddSibling(added))

		require.Equal(t, []string{"a", "b"}, elementNames(t, parent), "the parent's child list is unchanged")
		require.Equal(t, b, parent.LastChild())
		require.Equal(t, helium.Node(added), y.NextSibling(), "the append joins the anchor's own chain")
		require.Equal(t, helium.Node(y), added.PrevSibling())
		require.Nil(t, added.Parent(), "that chain has no owner to assign as parent")
	})

	t.Run("hand-written edge on a node another document owns", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDefaultDocument()
		other := helium.NewDefaultDocument()
		parent := mustCreateElement(t, doc, "parent")
		a := mustCreateElement(t, doc, "a")
		// foreign is owned by another document but lives in this one's chain, so
		// the write below lands on a node whose own document is not the one that
		// must stop trusting its tail record.
		foreign := mustCreateElement(t, other, "foreign")
		c := mustCreateElement(t, doc, "c")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(foreign))
		require.NoError(t, parent.AddChild(c))
		require.Equal(t, c, parent.LastChild())

		// Cut c out of the chain through the foreign node. c stays recorded as
		// the tail, still claiming parent and still ending its own chain.
		helium.UnsafeSetNextSibling(foreign, nil)

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, a.AddSibling(added))

		require.Equal(t, []string{"a", "foreign", nameAdded}, elementNames(t, parent), "the append lands at the end of the reachable chain")
		require.Equal(t, helium.Node(foreign), added.PrevSibling())
		require.Equal(t, added, parent.LastChild())
		require.Nil(t, c.NextSibling(), "the excised node keeps its own links")
	})

	t.Run("parent claim by a node another document owns", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDefaultDocument()
		other := helium.NewDefaultDocument()
		parent := mustCreateElement(t, doc, "parent")
		a := mustCreateElement(t, doc, "a")
		b := mustCreateElement(t, doc, "b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		// The claimant belongs to another document and is attached nowhere, so
		// only the parent it claims names the document whose chain is at stake.
		claimant := mustCreateElement(t, other, "claimant")
		helium.UnsafeSetParent(claimant, parent)
		trailer := mustCreateElement(t, doc, "trailer")
		require.NoError(t, claimant.AddSibling(trailer))
		require.Equal(t, trailer, parent.LastChild(), "the append records a tail off the child list")

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, a.AddSibling(added))

		require.Equal(t, []string{"a", "b", nameAdded}, elementNames(t, parent), "the append lands at the end of the reachable chain")
		require.Equal(t, helium.Node(b), added.PrevSibling())
		require.Equal(t, added, parent.LastChild())
		require.Nil(t, trailer.NextSibling(), "the off-chain chain is untouched")
	})

	t.Run("stale tail behind a hand-linked node", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDefaultDocument()
		parent := mustCreateElement(t, doc, "parent")
		a := mustCreateElement(t, doc, "a")
		b := mustCreateElement(t, doc, "b")
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		// d is linked in by hand after the recorded tail, so parent.lastChild is
		// stale: b is no longer the end of the chain.
		d := mustCreateElement(t, doc, "d")
		helium.UnsafeSetNextSibling(b, d)
		helium.UnsafeSetPrevSibling(d, b)
		helium.UnsafeSetParent(d, parent)

		added := mustCreateElement(t, doc, nameAdded)
		require.NoError(t, a.AddSibling(added))

		require.Equal(t, []string{"a", "b", "d", "added"}, elementNames(t, parent), "the append lands past the stale record")
		require.Equal(t, helium.Node(d), added.PrevSibling())
		require.Equal(t, added, parent.LastChild(), "and the stale record is repaired")
	})
}

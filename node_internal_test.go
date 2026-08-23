package helium

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

func TestCrossDocumentMove(t *testing.T) {
	// A slab-backed node moved from document A into document B must not be
	// overwritten after A.Free() recycles its slab chunks into the global pool:
	// A.Free is a no-op once a node escaped, so the moved node keeps its content.
	t.Run("subtree survives the source Free", func(t *testing.T) {
		a := NewDocument("1.0", "", StandaloneImplicitNo)
		moved, err := a.CreateElement("moved")
		require.NoError(t, err)
		txt := a.CreateText([]byte("ORIGINAL-CONTENT"))
		require.NoError(t, moved.AddChild(txt))

		b := NewDocument("1.0", "", StandaloneImplicitNo)
		broot, err := b.CreateElement("broot")
		require.NoError(t, err)
		require.NoError(t, b.AddChild(broot))

		// Move the subtree from A into B. This must mark A as having escaped storage.
		require.NoError(t, broot.AddChild(moved))
		require.True(t, a.slabEscaped, "moving a node into another document must mark the source as escaped")

		// Free A, then aggressively allocate in a fresh document to reuse any chunk
		// A might have returned to the pool. With the fix A returned nothing.
		a.Free()
		c := NewDocument("1.0", "", StandaloneImplicitNo)
		for range 512 {
			e, err := c.CreateElement("OVERWRITE")
			require.NoError(t, err)
			tx := c.CreateText([]byte("XXXXXXXXXXXXXXXX"))
			require.NoError(t, e.AddChild(tx))
		}

		require.Equal(t, "moved", moved.Name(), "moved element struct was overwritten by a reused slab chunk")
		require.Equal(t, "ORIGINAL-CONTENT", string(txt.Content()), "moved text content bytes were overwritten by reused slab storage")
	})

	// A routed-attribute move across documents: an element in document A owns an
	// attribute, and an element in document B adopts it via elem.AddChild(attr).
	// The attribute must leave A, land on B with its value, mark A as having
	// escaped storage, and keep its value after A.Free() recycles its slab chunks
	// under allocation pressure. This exercises the noteCrossDocumentEscape path
	// for a property-list (attribute) move, distinct from the child-list moves
	// covered above. The attribute is built with a.CreateAttribute so both its
	// struct and its value text are slab-backed by A, and the churn below
	// allocates through c.CreateAttribute so the matching attribute/text slabs are
	// actually redrawn from the pool — a wrongly recycled chunk zeroes the moved
	// attribute and fails the assertions.
	t.Run("attribute move survives the source Free", func(t *testing.T) {
		a := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
		aroot, err := a.CreateElement("aroot")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(aroot))
		attr, err := a.CreateAttribute("moved", "CROSSDOC-VALUE", nil)
		require.NoError(t, err)
		require.NoError(t, aroot.AddChild(attr))

		b := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
		broot, err := b.CreateElement("broot")
		require.NoError(t, err)
		require.NoError(t, b.AddChild(broot))

		// The cross-document routed-attribute move under test.
		require.NoError(t, broot.AddChild(attr))
		require.True(t, a.slabEscaped, "moving an attribute into another document must mark the source escaped")

		// Gone from the old owner.
		_, ok := aroot.GetAttribute("moved")
		require.False(t, ok, "attribute still reported by old owner GetAttribute")
		require.Empty(t, aroot.Attributes(), "old owner still holds the moved attribute")

		// Present on the new owner with the right value.
		got, ok := broot.GetAttribute("moved")
		require.True(t, ok, "attribute not reachable on new owner")
		require.Equal(t, "CROSSDOC-VALUE", got)

		// Both documents serialize correctly.
		aout, err := WriteString(a)
		require.NoError(t, err)
		require.NotContains(t, aout, "moved", "doc A still serializes the moved attribute")
		bout, err := WriteString(b)
		require.NoError(t, err)
		require.Contains(t, bout, `moved="CROSSDOC-VALUE"`, "doc B does not serialize the moved attribute")

		// Slab-safety: free A, churn slab allocations (element, attribute, text,
		// and text-content) in a fresh document to redraw any recycled chunks from
		// the pool, then confirm the moved attribute's storage is intact.
		a.Free()
		c := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
		for range 512 {
			e, err := c.CreateElement("OVERWRITE")
			require.NoError(t, err)
			ka, err := c.CreateAttribute("k", "XXXXXXXXXXXXXXXX", nil)
			require.NoError(t, err)
			require.NoError(t, e.AddChild(ka))
		}

		got2, ok := broot.GetAttribute("moved")
		require.True(t, ok, "attribute lost after A.Free()+churn; slab storage was recycled")
		require.Equal(t, "CROSSDOC-VALUE", got2, "attribute value overwritten by reused slab storage")
		bout2, err := WriteString(b)
		require.NoError(t, err)
		require.Contains(t, bout2, `moved="CROSSDOC-VALUE"`, "doc B post-Free serialization lost the attribute")
	})

	// Moving a node within one document is not a cross-document escape, so the
	// flag stays clear and Free keeps recycling.
	t.Run("same-document move does not escape", func(t *testing.T) {
		d := NewDocument("1.0", "", StandaloneImplicitNo)
		root, err := d.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, d.AddChild(root))
		a, err := d.CreateElement("a")
		require.NoError(t, err)
		b, err := d.CreateElement("b")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(b))

		// Re-parent a under b, all within document d.
		require.NoError(t, b.AddChild(a))
		require.False(t, d.slabEscaped, "an intra-document move must not mark the document as escaped")
	})

	// The common path: a single-document parse never marks the document as
	// escaped, so Free still recycles its slab chunks.
	t.Run("plain parse does not escape", func(t *testing.T) {
		src := []byte(`<?xml version="1.0"?><root xmlns:x="urn:x"><a x:k="v">hi &amp; bye</a><b/></root>`)
		doc, err := NewParser().Parse(context.Background(), src)
		require.NoError(t, err)
		require.False(t, doc.slabEscaped, "a plain single-document parse must not mark the document as escaped")
	})
}

// recycleNamespaceSlab allocates enough namespaces in a fresh document to draw
// chunks back out of the shared pool and overwrite any chunk a freed document
// returned to it.
func recycleNamespaceSlab(t *testing.T) {
	t.Helper()
	c := NewDocument("1.0", "", StandaloneImplicitNo)
	for range 2 * slabSize {
		_, err := c.CreateNamespace("q", "urn:overwrite")
		require.NoError(t, err)
	}
}

func TestAddNamespaceDecl(t *testing.T) {
	// Case 1 (append): AddNamespaceDecl retains a foreign slab-backed Namespace,
	// so the source document's Free must not recycle its slab out from under the
	// retained decl.
	t.Run("cross-document append survives the source Free", func(t *testing.T) {
		a := NewDocument("1.0", "", StandaloneImplicitNo)
		el, err := a.CreateElement("el")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(el))

		b := NewDocument("1.0", "", StandaloneImplicitNo)
		ns, err := b.CreateNamespace("p", "urn:new")
		require.NoError(t, err)

		require.NoError(t, el.AddNamespaceDecl(ns)) // case 1: no existing entry for "p" -> append (retains ns)
		require.True(t, b.slabEscaped, "retaining a foreign slab-backed namespace must mark the source escaped")

		b.Free()
		recycleNamespaceSlab(t)

		require.Equal(t, "p", ns.Prefix(), "retained namespace prefix was overwritten by a reused slab chunk")
		require.Equal(t, "urn:new", ns.URI(), "retained namespace URI was overwritten by a reused slab chunk")
		got := el.Namespaces()
		require.Len(t, got, 1)
		require.Equal(t, "p", got[0].Prefix())
		require.Equal(t, "urn:new", got[0].URI())
	})

	// Case 3 (collapse): AddNamespaceDecl replaces an existing same-prefix slot
	// with the caller's foreign slab-backed Namespace, which is likewise retained
	// and must survive the source document's Free.
	t.Run("cross-document collapse survives the source Free", func(t *testing.T) {
		a := NewDocument("1.0", "", StandaloneImplicitNo)
		el, err := a.CreateElement("el")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(el))
		require.NoError(t, el.DeclareNamespace("p", "urn:old")) // A-owned slot

		b := NewDocument("1.0", "", StandaloneImplicitNo)
		ns, err := b.CreateNamespace("p", "urn:new")
		require.NoError(t, err)

		require.NoError(t, el.AddNamespaceDecl(ns)) // case 3: existing "p" at a different URI -> collapse (retains ns)
		require.True(t, b.slabEscaped, "collapsing in a foreign slab-backed namespace must mark the source escaped")

		b.Free()
		recycleNamespaceSlab(t)

		require.Equal(t, "p", ns.Prefix(), "retained namespace prefix was overwritten by a reused slab chunk")
		require.Equal(t, "urn:new", ns.URI(), "retained namespace URI was overwritten by a reused slab chunk")
		got := el.Namespaces()
		require.Len(t, got, 1)
		require.Equal(t, "p", got[0].Prefix())
		require.Equal(t, "urn:new", got[0].URI())
	})

	// A namespace owned by the same document as the receiver is not a
	// cross-document escape, so the flag stays clear and Free keeps recycling.
	t.Run("same-document decl does not escape", func(t *testing.T) {
		a := NewDocument("1.0", "", StandaloneImplicitNo)
		el, err := a.CreateElement("el")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(el))
		ns, err := a.CreateNamespace("p", "urn:x")
		require.NoError(t, err)

		require.NoError(t, el.AddNamespaceDecl(ns)) // same document -> no escape
		require.False(t, a.slabEscaped, "a same-document namespace decl must not mark escape")
	})

	// A case-2 no-op (an existing declaration at the same URI keeps its slot; the
	// caller's object is not retained) must not mark the source document escaped.
	t.Run("cross-document no-op does not escape", func(t *testing.T) {
		a := NewDocument("1.0", "", StandaloneImplicitNo)
		el, err := a.CreateElement("el")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(el))
		require.NoError(t, el.DeclareNamespace("p", "urn:same")) // existing A-owned slot

		b := NewDocument("1.0", "", StandaloneImplicitNo)
		ns, err := b.CreateNamespace("p", "urn:same") // same URI -> case 2 no-op
		require.NoError(t, err)

		require.NoError(t, el.AddNamespaceDecl(ns))
		require.False(t, b.slabEscaped, "a same-URI no-op must not mark the source escaped")
	})

	// The per-prefix dedup scan in DeclareNamespace/AddNamespaceDecl tolerates a
	// nil nsDefs slot. The public AddNamespaceDecl rejects a nil ns with
	// ErrNilNode, so a nil entry can only arise from a direct in-package field
	// write; the scan must skip it and never dereference it.
	t.Run("nil nsDefs entry is skipped", func(t *testing.T) {
		doc := NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		root.nsDefs = append(root.nsDefs, nil)

		require.NotPanics(t, func() {
			require.NoError(t, root.DeclareNamespace("p", "urn:p"))
		})
		require.NotPanics(t, func() {
			require.NoError(t, root.AddNamespaceDecl(NewNamespace("q", "urn:q")))
		})
	})
}

func TestSetNamespace(t *testing.T) {
	// The active-namespace retention path: SetNamespace installs a foreign
	// slab-backed Namespace as the node's active namespace, so the source
	// document's Free must not recycle its slab out from under it.
	t.Run("cross-document namespace survives the source Free", func(t *testing.T) {
		a := NewDocument("1.0", "", StandaloneImplicitNo)
		el, err := a.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(el))

		b := NewDocument("1.0", "", StandaloneImplicitNo)
		ns, err := b.CreateNamespace("p", "urn:original")
		require.NoError(t, err)

		el.SetNamespace(ns)
		require.True(t, b.slabEscaped, "installing a foreign slab-backed active namespace must mark the source escaped")

		b.Free()
		recycleNamespaceSlab(t)

		require.Equal(t, "p", ns.Prefix(), "retained active namespace prefix was overwritten by a reused slab chunk")
		require.Equal(t, "urn:original", ns.URI(), "retained active namespace URI was overwritten by a reused slab chunk")
		require.Equal(t, "p:root", el.Name())
	})

	// CreateElementNS reaches SetNamespace: an element created in one document
	// with a namespace owned by another must keep its prefix/URI after the source
	// document is freed and its slab is recycled. The observable serialization
	// stays <p:root>, and never mutates to the reused binding.
	t.Run("CreateElementNS namespace survives the source Free", func(t *testing.T) {
		dest := NewDocument("1.0", "", StandaloneImplicitNo)
		src := NewDocument("1.0", "", StandaloneImplicitNo)
		ns, err := src.CreateNamespace("p", "urn:original")
		require.NoError(t, err)

		el, err := dest.CreateElementNS("root", ns)
		require.NoError(t, err)
		require.True(t, src.slabEscaped, "CreateElementNS retaining a foreign namespace must mark the source escaped")
		require.NoError(t, dest.SetDocumentElement(el))

		src.Free()
		recycleNamespaceSlab(t)

		require.Equal(t, "p:root", el.Name(), "retained namespace mutated after source Free")
		s, err := WriteString(el)
		require.NoError(t, err)
		require.Contains(t, s, "p:root")
		require.NotContains(t, s, "urn:overwrite")
	})

	// A same-document active namespace is not a cross-document escape, so the flag
	// stays clear and Free keeps recycling.
	t.Run("same-document namespace does not escape", func(t *testing.T) {
		a := NewDocument("1.0", "", StandaloneImplicitNo)
		el, err := a.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(el))
		ns, err := a.CreateNamespace("p", "urn:x")
		require.NoError(t, err)

		el.SetNamespace(ns)
		require.False(t, a.slabEscaped, "a same-document active namespace must not mark escape")
	})
}

func TestChildReaches(t *testing.T) {
	// The child-pointer reachability search has no depth cap: a target reachable
	// only past the old 4096-node cap is still found. A capped search would fail
	// OPEN (return false = "not reachable"), letting a cycle deeper than the cap
	// slip through the insertion guard.
	t.Run("no depth cap", func(t *testing.T) {
		doc := NewDefaultDocument()

		// A single-child chain first -> n -> n -> ... -> deep, deeper than the
		// removed 4096 cap. childReaches must descend the whole chain.
		const depth = 5000
		first, err := doc.CreateElement("n0")
		require.NoError(t, err)
		prev := first
		for i := 1; i <= depth; i++ {
			cur, err := doc.CreateElement("n")
			require.NoError(t, err)
			require.NoError(t, prev.AddChild(cur))
			prev = cur
		}

		require.True(t, childReaches(first, prev.baseDocNode()),
			"a target %d levels deep must be found — the search must not cap depth and fail open", depth)

		// A node NOT in the subtree is not reachable.
		outside, err := doc.CreateElement("outside")
		require.NoError(t, err)
		require.False(t, childReaches(first, outside.baseDocNode()),
			"a node outside the subtree must not be reported reachable")
	})

	// The search must terminate when a node's child list has a cyclic sibling
	// pointer. The popped-node visited set alone does not bound the inner sibling
	// enumeration, so a self-referential or 2-cycle sibling link would spin forever
	// without the per-list sibling guard.
	t.Run("terminates on a cyclic sibling list", func(t *testing.T) {
		t.Run("self-cycle", func(t *testing.T) {
			doc := NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			c, err := doc.CreateElement("c")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(c))

			// Corrupt the sibling list into a self-cycle: c.next = c.
			UnsafeSetNextSibling(c, c)

			outside, err := doc.CreateElement("outside")
			require.NoError(t, err)
			require.False(t, childReaches(parent, outside.baseDocNode()),
				"a target not in the cyclic list must terminate and report false")
			require.True(t, childReaches(parent, c.baseDocNode()),
				"a node in the cyclic list must still be found")
		})

		t.Run("two-cycle", func(t *testing.T) {
			doc := NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			c1, err := doc.CreateElement("c1")
			require.NoError(t, err)
			c2, err := doc.CreateElement("c2")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(c1))
			require.NoError(t, parent.AddChild(c2))

			// Close a 2-cycle: c1 -> c2 -> c1.
			UnsafeSetNextSibling(c2, c1)

			outside, err := doc.CreateElement("outside")
			require.NoError(t, err)
			require.False(t, childReaches(parent, outside.baseDocNode()),
				"a target not in the cyclic list must terminate and report false")
		})
	})
}

func TestWalkCycleGuards(t *testing.T) {
	// Walk returns ErrWalkCycle, and never loops forever, on a child-pointer
	// cycle that the guarded insertion API refuses to build: an entity reference's
	// shared Entity child links back to the reference (ent.firstChild = ref,
	// ref.firstChild = ent). The cycle is closed through the lower-level docnode
	// links, bypassing AddChild.
	t.Run("entity child cycle", func(t *testing.T) {
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
	})

	// Walk terminates, and never spins forever, on a sibling cycle LONGER
	// than one node: a parent whose two children form a 2-cycle a -> b -> a. The
	// active-path guard alone does not catch this — each child is popped off the
	// stack before its next sibling is examined — so the per-frame seenChildren set
	// must return ErrWalkCycle.
	t.Run("sibling cycle", func(t *testing.T) {
		doc := NewDefaultDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		a, err := doc.CreateElement("a")
		require.NoError(t, err)
		b, err := doc.CreateElement("b")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(a))
		require.NoError(t, parent.AddChild(b))

		// Close a 2-cycle in the sibling list: a -> b -> a.
		UnsafeSetNextSibling(b, a)

		err = Walk(parent, NodeWalkerFunc(func(Node) error { return nil }))
		require.ErrorIs(t, err, ErrWalkCycle,
			"Walk must detect the sibling cycle and return ErrWalkCycle instead of hanging")
	})
}

// The aggregating Content() terminates on a pure child-pointer cycle NOT routed
// through a terminating Entity: a -> b -> a, built with the low-level link
// primitives that bypass the guarded AddChild. Without the active-path guard the
// container recursion would recurse forever (stack overflow); with it, the
// back-edge into a (already on the active path) is skipped and the sibling text
// is still aggregated.
func TestContentCycleGuard(t *testing.T) {
	doc := NewDefaultDocument()
	a, err := doc.CreateElement("a")
	require.NoError(t, err)
	txt := doc.CreateText([]byte("x"))
	b, err := doc.CreateElement("b")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(txt))
	require.NoError(t, a.AddChild(b))

	// Close a child-pointer cycle a -> b -> a: b's only child is a (parent link
	// included so it is a genuine owned child, invisible to the owned-boundary
	// advance and caught only by the active-path guard).
	setFirstChild(b, a)
	setLastChild(b, a)
	UnsafeSetParent(a, b)

	require.Equal(t, []byte("x"), a.Content(),
		"Content must terminate on the child-pointer back-edge and still aggregate the text sibling")
}

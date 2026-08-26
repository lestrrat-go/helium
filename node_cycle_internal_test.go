package helium

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// referenceWouldCreateCycle is the unconditional form of the guard: it always
// walks parent's ancestor chain, and descends cur's children whenever cur has
// any. wouldCreateCycle must answer identically for every input — the claimant
// search is a cost optimization and is not allowed to change a single verdict.
func referenceWouldCreateCycle(parent, cur Node) bool {
	cdn := cur.baseDocNode()
	for anc := parent; anc != nil; anc = anc.Parent() {
		if anc.baseDocNode() == cdn {
			return true
		}
	}
	if parent == nil || cdn.firstChild == nil {
		return false
	}
	return childReaches(cur, parent.baseDocNode())
}

// cycleCase names one (parent, cur) configuration and the verdict both guards
// must reach on it.
type cycleCase struct {
	name string
	// build returns the insertion point and the operand.
	build func(t *testing.T) (Node, Node)
	want  bool
}

// newCycleDoc builds an empty document for a case fixture.
func newCycleDoc(t *testing.T) *Document {
	t.Helper()
	return NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
}

// newCycleChain builds a parent chain of the given depth and returns its
// deepest element together with the root the chain hangs from.
func newCycleChain(t *testing.T, doc *Document, depth int) (Node, Node) {
	t.Helper()
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	cur := MutableNode(root)
	for i := range depth {
		e, err := doc.CreateElement(fmt.Sprintf("e%d", i))
		require.NoError(t, err)
		require.NoError(t, cur.AddChild(e))
		cur = e
	}
	return root, cur
}

// offChainClaimFixture builds the shape the guarded paths create themselves: a
// childless entity that a detached element still names as its parent. It
// returns that element and the entity it claims.
func offChainClaimFixture(t *testing.T) (*Element, *Entity) {
	t.Helper()

	doc, err := NewParser().Parse(t.Context(), []byte(`<!DOCTYPE d [<!ENTITY e "xy">]><d a="&e;"/>`))
	require.NoError(t, err)

	ent, ok := doc.GetEntity("e")
	require.True(t, ok)
	require.NotNil(t, ent.FirstChild(), "the attribute-value expansion is the entity's child")
	require.Nil(t, ent.LastChild(), "and it was recorded without a lastChild")

	claimant, err := doc.CreateElement("claimant")
	require.NoError(t, err)
	expansion, ok := ent.FirstChild().(MutableNode)
	require.True(t, ok)
	require.NoError(t, expansion.Replace(claimant))
	require.Equal(t, Node(ent), claimant.Parent())

	// The append takes the empty-parent branch, which overwrites firstChild and
	// detaches claimant while it goes on naming the entity as its parent.
	// Unlinking the replacement empties the child list again, which is exactly
	// the shape the claimant search is consulted on.
	filler := doc.CreateComment([]byte("filler"))
	require.NoError(t, ent.AddChild(filler))
	UnlinkNode(filler)
	require.Nil(t, ent.FirstChild())
	require.Nil(t, ent.LastChild())
	require.Equal(t, Node(ent), claimant.Parent(), "the detached child still claims the entity")

	return claimant, ent
}

// cycleCases enumerates the configurations the two guards are compared on.
// Each one is a shape the claimant search must classify from cur's own slots
// rather than from parent's depth.
func cycleCases() []cycleCase {
	return []cycleCase{
		{
			name: "childless operand under an unrelated parent",
			want: false,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				_, deep := newCycleChain(t, doc, 8)
				fresh, err := doc.CreateElement("fresh")
				require.NoError(t, err)
				return deep, fresh
			},
		},
		{
			name: "operand inserted under itself",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				return e, e
			},
		},
		{
			name: "operand is a deep ancestor of the parent",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				root, deep := newCycleChain(t, doc, 8)
				return deep, root
			},
		},
		{
			name: "operand is the immediate parent of the insertion point",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				parent, err := doc.CreateElement("parent")
				require.NoError(t, err)
				kid, err := doc.CreateElement("kid")
				require.NoError(t, err)
				require.NoError(t, parent.AddChild(kid))
				return kid, parent
			},
		},
		{
			name: "childless operand whose attribute is the insertion point",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, e.SetAttribute("a", "v"))
				attrs := e.Attributes()
				require.Len(t, attrs, 1)
				return attrs[0], e
			},
		},
		{
			name: "childless operand whose attribute text is the insertion point",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, e.SetAttribute("a", "v"))
				attrs := e.Attributes()
				require.Len(t, attrs, 1)
				text := attrs[0].FirstChild()
				require.NotNil(t, text, "attribute carries its value as a text child")
				return text, e
			},
		},
		{
			name: "operand carrying attributes under an unrelated parent",
			want: false,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				_, deep := newCycleChain(t, doc, 8)
				fresh, err := doc.CreateElement("fresh")
				require.NoError(t, err)
				require.NoError(t, fresh.SetAttribute("a", "v"))
				require.NoError(t, fresh.SetAttribute("b", "w"))
				return deep, fresh
			},
		},
		{
			name: "operand with children containing the insertion point",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				outer, err := doc.CreateElement("outer")
				require.NoError(t, err)
				inner, err := doc.CreateElement("inner")
				require.NoError(t, err)
				require.NoError(t, outer.AddChild(inner))
				return inner, outer
			},
		},
		{
			name: "entity reference reachable only through a foreign child link",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				dtd, err := doc.CreateInternalSubset("root", "", "")
				require.NoError(t, err)
				ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
				require.NoError(t, err)
				ref, err := doc.CreateReference("e")
				require.NoError(t, err)
				require.Equal(t, Node(ent), ref.FirstChild())
				return ent, ref
			},
		},
		{
			name: "document operand whose internal subset holds the insertion point",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				dtd, err := doc.CreateInternalSubset("root", "", "")
				require.NoError(t, err)
				return dtd, doc
			},
		},
		{
			name: "childless operand holding an off-chain parent claim",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				claimant, ent := offChainClaimFixture(t)
				return claimant, ent
			},
		},
		{
			name: "document operand whose detached internal subset is the insertion point",
			want: false,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				dtd, err := doc.CreateInternalSubset("root", "", "")
				require.NoError(t, err)

				// unlinkNode clears the subset's parent and empties the
				// document's child list, but leaves Document.intSubset pointing
				// at it. The subset no longer names the document, so no ancestor
				// chain runs through it and linking the document under it is not
				// a cycle.
				UnlinkNode(dtd)
				require.Nil(t, doc.FirstChild())
				require.Nil(t, dtd.Parent())
				require.Equal(t, dtd, doc.IntSubset())

				return dtd, doc
			},
		},
		{
			name: "document operand whose off-list internal subset is the insertion point",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				dtd, err := doc.CreateInternalSubset("root", "", "")
				require.NoError(t, err)

				// The same shape with the claim intact: the subset is off the
				// document's child list, so the claimant search is consulted,
				// and it still names the document, so the chain the ancestor
				// walk would step is really there.
				UnlinkNode(dtd)
				unsafeSetParent(dtd, doc)
				require.Nil(t, doc.FirstChild())

				return dtd, doc
			},
		},
		{
			name: "childless operand claimed only through a foreign child of its attribute",
			want: false,
			build: func(t *testing.T) (Node, Node) {
				doc, err := NewParser().Parse(t.Context(), []byte(
					`<!DOCTYPE root [<!ENTITY e "val">]><root><child a="&e;"/></root>`))
				require.NoError(t, err)

				ent, ok := doc.GetEntity("e")
				require.True(t, ok)

				root := doc.DocumentElement()
				require.NotNil(t, root)
				elem, ok := root.FirstChild().(*Element)
				require.True(t, ok)
				require.Nil(t, elem.FirstChild())

				// The attribute value expands to an entity reference whose child
				// is the shared entity, and that entity's parent is the DTD. The
				// reference does not own it, so it is on no chain of parent
				// pointers leading down from the element.
				attrs := elem.Attributes()
				require.Len(t, attrs, 1)
				ref := attrs[0].FirstChild()
				require.Equal(t, EntityRefNode, ref.Type())
				require.Equal(t, Node(ent), ref.FirstChild())
				require.Equal(t, DTDNode, ent.Parent().Type())

				return ent, elem
			},
		},
		{
			name: "document-less operand holding an off-chain parent claim",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				claimant, ent := offChainClaimFixture(t)

				// The claim was recorded on the owning document. Taking that
				// document away leaves the entity with nowhere document-shaped
				// to read it from, so the record has to travel with it.
				ent.SetTreeDoc(nil)
				require.Nil(t, ent.OwnerDocument())
				require.Equal(t, Node(ent), claimant.Parent())

				return claimant, ent
			},
		},
		{
			name: "document operand claimed from its own lastChild alone",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)

				// CopyExtSubset gives the copy the destination document as its
				// parent and leaves it off the child list. Appending through it
				// writes the document's lastChild while firstChild stays nil, so
				// the appended node claims the document from a slot the claimant
				// search does not enumerate.
				src := NewDefaultDocument()
				srcDTD := newDTD()
				srcDTD.name = "root"
				src.extSubset = srcDTD
				CopyExtSubset(src, doc)
				ext := doc.ExtSubset()
				require.Equal(t, Node(doc), ext.Parent())
				require.Nil(t, doc.FirstChild())

				claimant, err := doc.CreateElement("claimant")
				require.NoError(t, err)
				require.NoError(t, ext.AddSibling(claimant))
				require.Equal(t, Node(doc), claimant.Parent())
				require.Nil(t, doc.FirstChild())
				require.Equal(t, Node(claimant), doc.LastChild())

				return claimant, doc
			},
		},
		{
			name: "childless operand claimed through an attribute two hops down",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				cur, err := doc.CreateElement("cur")
				require.NoError(t, err)
				require.NoError(t, cur.SetAttribute("a", "v"))
				attrs := cur.Attributes()
				require.Len(t, attrs, 1)

				// The chain leaves the child lists twice: cur is claimed by its
				// attribute, and the element under that attribute is claimed by
				// an attribute of its own. A descent that enumerates slots only
				// at the operand stops at the second hop, where the child list
				// is empty and the claimant sits in properties.
				mid, err := doc.CreateElement("mid")
				require.NoError(t, err)
				require.NoError(t, attrs[0].AddChild(mid))
				require.NoError(t, mid.SetAttribute("p", "x"))
				deep := mid.Attributes()
				require.Len(t, deep, 1)

				require.Nil(t, cur.FirstChild(), "the operand is settled from its slots, not its child list")
				return deep[0], cur
			},
		},
		{
			name: "document operand claimed through an attribute under its internal subset",
			want: true,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				dtd, err := doc.CreateInternalSubset("root", "", "")
				require.NoError(t, err)

				// CreateInternalSubset also lists the subset as a child, and the
				// claimant search is consulted only on an operand whose child
				// list is empty. Unlink it and restore the claim, so the
				// document is reachable only through its intSubset slot.
				UnlinkNode(dtd)
				unsafeSetParent(dtd, doc)
				require.Nil(t, doc.FirstChild())

				elem, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, dtd.AddChild(elem))
				require.NoError(t, elem.SetAttribute("a", "v"))
				attrs := elem.Attributes()
				require.Len(t, attrs, 1)

				return attrs[0], doc
			},
		},
		{
			name: "childless operand with no parent at all",
			want: false,
			build: func(t *testing.T) (Node, Node) {
				doc := newCycleDoc(t)
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				return nil, e
			},
		},
	}
}

// TestWouldCreateCycleMatchesUnconditionalWalk is the guard's contract test.
// The claimant search answers from the operand's own claimant slots instead of
// walking the insertion point's ancestors, so every shape where those two
// sources could disagree is compared against the unconditional form here.
func TestWouldCreateCycleMatchesUnconditionalWalk(t *testing.T) {
	t.Parallel()

	for _, tc := range cycleCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parent, cur := tc.build(t)
			want := referenceWouldCreateCycle(parent, cur)
			require.Equal(t, tc.want, want, "the reference guard must reach the verdict the case declares")
			require.Equal(t, want, wouldCreateCycle(parent, cur), "the claimant search must not change a verdict")
		})
	}
}

// parentProbe stands in for an insertion point and records every step the
// ancestor walk takes through it. The claimant search settles an operand from
// the operand's own slots, so a walk step here is the optimization not firing.
type parentProbe struct {
	docnode
	parentCalls int
}

// Parent counts one ancestor-walk step and ends the chain.
func (p *parentProbe) Parent() Node {
	p.parentCalls++
	return nil
}

// TestClaimantSearchSkipsTheAncestorWalk pins the property the optimization
// exists for: an operand that nothing can name as its parent is settled
// without consulting the insertion point's ancestors at all. The probe counts
// those steps, so a regression to the unconditional walk fails here instead of
// only showing up as a slower benchmark.
func TestClaimantSearchSkipsTheAncestorWalk(t *testing.T) {
	t.Parallel()

	t.Run("childless operand", func(t *testing.T) {
		t.Parallel()

		doc := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
		fresh, err := doc.CreateElement("fresh")
		require.NoError(t, err)

		probe := &parentProbe{}
		require.False(t, wouldCreateCycle(probe, fresh))
		require.Zero(t, probe.parentCalls, "a childless operand must not start the ancestor walk")
	})

	t.Run("operand claimed only by its own attributes", func(t *testing.T) {
		t.Parallel()

		doc := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
		withAttr, err := doc.CreateElement("withattr")
		require.NoError(t, err)
		require.NoError(t, withAttr.SetAttribute("a", "v"))

		probe := &parentProbe{}
		require.False(t, wouldCreateCycle(probe, withAttr))
		require.Zero(t, probe.parentCalls, "an attribute-claimed operand is settled from its attribute list")
	})
}

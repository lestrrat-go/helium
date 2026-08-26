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

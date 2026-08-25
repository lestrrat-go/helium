package helium

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// legacyWouldCreateCycle is a TEST-ONLY reference copy of the insertion cycle
// guard as it stood before the childless-operand fast path was introduced: an
// UNCONDITIONAL ancestor walk over parent's parent chain, plus the child-pointer
// reachability search for an operand that has children. It is never called by
// production code and exists solely so TestWouldCreateCycleDifferential can
// compare the shipping guard's verdict against the verdict the depth-
// proportional walk gave, case by case. Keep it byte-faithful to the pre-change
// implementation: editing it to agree with wouldCreateCycle would silently turn
// the differential test back into a restatement of the shipping guard.
func legacyWouldCreateCycle(parent, cur Node) bool {
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

func newCycleCaseDocument() *Document {
	return NewDocument("1.0", "utf-8", StandaloneExplicitNo)
}

// cycleCase is one adversarial insertion. build assembles the tree and returns
// the two operands the guard is handed for the case's FINAL insertion — for
// AddChild that is (receiver, argument), for AddSibling and Replace it is
// (receiver's parent, argument), matching addChildPreflight,
// addSiblingPreflight and Replace — together with a closure performing that
// insertion through the public API. build must not perform the insertion
// itself: the harness evaluates both guards against the untouched tree first.
type cycleCase struct {
	name       string
	build      func(t *testing.T) (Node, Node, func() error)
	wantCyclic bool
	// outsideGuardContract marks a case whose tree was built behind the safe
	// mutation API's back, where the shipping guard deliberately no longer
	// agrees with the reference implementation. The harness then requires the
	// verdicts to DIFFER in exactly the documented direction, so a change to
	// either side still fails this test.
	outsideGuardContract bool
}

// TestWouldCreateCycleDifferential runs every adversarial case through BOTH the
// shipping guard and legacyWouldCreateCycle, the pre-change reference
// implementation, and requires their verdicts to be identical. The hand-written
// wantCyclic column is kept as a secondary assertion; it cannot establish
// equivalence on its own, because a divergence both implementations were never
// compared on is exactly what a hand-written expectation misses.
//
// The cases include the two attribute-pocket shapes (a childless *Attribute and
// a childless entity reference living inside an attribute value) that a
// childless-operand fast path must still resolve through the Element properties
// chain, and one case built through UnsafeSetParent, the single shape where the
// two implementations are documented to differ.
func TestWouldCreateCycleDifferential(t *testing.T) {
	tests := []cycleCase{
		{
			name: "self AddChild",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				a, err := doc.CreateElement("a")
				require.NoError(t, err)
				return a, a, func() error { return a.AddChild(a) }
			},
			wantCyclic: true,
		},
		{
			name: "leaf.AddChild(ancestor 50 up)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				root, err := doc.CreateElement("root")
				require.NoError(t, err)
				require.NoError(t, doc.SetDocumentElement(root))
				cur := root
				var leaf *Element
				for range 50 {
					e, err := doc.CreateElement("e")
					require.NoError(t, err)
					require.NoError(t, cur.AddChild(e))
					cur = e
					leaf = e
				}
				return leaf, root, func() error { return leaf.AddChild(root) }
			},
			wantCyclic: true,
		},
		{
			name: "leaf.AddChild(document)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				root, err := doc.CreateElement("root")
				require.NoError(t, err)
				require.NoError(t, doc.SetDocumentElement(root))
				leaf, err := doc.CreateElement("leaf")
				require.NoError(t, err)
				require.NoError(t, root.AddChild(leaf))
				return leaf, doc, func() error { return leaf.AddChild(doc) }
			},
			wantCyclic: true,
		},
		{
			// The attribute-pocket case: attr has no children, so the fast path
			// must resolve it through Element.properties rather than assuming a
			// childless operand has no claimant.
			name: "attr.AddChild(childless owner)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				elem, err := doc.CreateElement("e")
				require.NoError(t, err)
				attr, err := doc.CreateAttribute("a", "", nil)
				require.NoError(t, err)
				require.NoError(t, elem.AddChild(attr))
				return attr, elem, func() error { return attr.AddChild(elem) }
			},
			wantCyclic: true,
		},
		{
			// The second attribute-pocket case: the claimant is a value child
			// inside the attribute (an entity reference), not the attribute
			// itself, so propertiesReach must also search each attribute's value
			// subtree.
			name: "attrValueRef.AddChild(childless owner)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				elem, err := doc.CreateElement("e")
				require.NoError(t, err)
				attr, err := doc.CreateAttribute("a", "x&foo;y", nil)
				require.NoError(t, err)
				require.NoError(t, elem.AddChild(attr))
				var ref Node
				for c := range Children(attr) {
					if c.Type() == EntityRefNode {
						ref = c
					}
				}
				require.NotNil(t, ref, "attribute value entity ref must be built")
				m, ok := ref.(MutableNode)
				require.True(t, ok)
				return ref, elem, func() error { return m.AddChild(elem) }
			},
			wantCyclic: true,
		},
		{
			name: "g.AddChild(p) [p has children]",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				g, err := doc.CreateElement("g")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				require.NoError(t, c.AddChild(g))
				return g, p, func() error { return g.AddChild(p) }
			},
			wantCyclic: true,
		},
		{
			// The foreign-link case: an entity reference's child is the shared
			// Entity node whose parent stays the DTD, so a cycle formed through
			// it is invisible to the ancestor walk and only childReaches can see
			// it. Both implementations run that same descent.
			name: "ent.AddChild(ref-with-foreign-child)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				dtd, err := doc.CreateDTD()
				require.NoError(t, err)
				ent, err := dtd.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
				require.NoError(t, err)
				ref, err := doc.CreateReference("foo")
				require.NoError(t, err)
				return ent, ref, func() error { return ent.AddChild(ref) }
			},
			wantCyclic: false,
		},
		{
			name: "plain AddChild",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				return p, c, func() error { return p.AddChild(c) }
			},
			wantCyclic: false,
		},
		{
			name: "AddSibling",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				s, err := doc.CreateElement("s")
				require.NoError(t, err)
				return c.Parent(), s, func() error { return c.AddSibling(s) }
			},
			wantCyclic: false,
		},
		{
			name: "AddChild(subtree)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				sub, err := doc.CreateElement("sub")
				require.NoError(t, err)
				sc, err := doc.CreateElement("sc")
				require.NoError(t, err)
				require.NoError(t, sub.AddChild(sc))
				return p, sub, func() error { return p.AddChild(sub) }
			},
			wantCyclic: false,
		},
		{
			name: "Replace",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				sc, err := doc.CreateElement("sc")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(sc))
				r, err := doc.CreateElement("r")
				require.NoError(t, err)
				return sc.Parent(), r, func() error { return sc.Replace(r) }
			},
			wantCyclic: false,
		},
		{
			name: "g.Replace(p)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				g, err := doc.CreateElement("g")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				require.NoError(t, c.AddChild(g))
				return g.Parent(), p, func() error { return g.Replace(p) }
			},
			wantCyclic: true,
		},
		{
			name: "g.AddSibling(p)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				g, err := doc.CreateElement("g")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				require.NoError(t, c.AddChild(g))
				return g.Parent(), p, func() error { return g.AddSibling(p) }
			},
			wantCyclic: true,
		},
		{
			// The documented divergence. UnsafeSetParent points parent's parent
			// pointer at cur without linking parent into cur's child list, so cur
			// stays a childless *Element with no attributes: it has no claimant in
			// any source the fast path consults, and the insertion is accepted
			// where the unconditional ancestor walk found cur one step up and
			// rejected it. A tree corrupted this way is outside the guard's
			// contract, as UnsafeSetParent's own documentation states.
			name: "AddChild(UnsafeSetParent-corrupted ancestor)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				parent, err := doc.CreateElement("parent")
				require.NoError(t, err)
				cur, err := doc.CreateElement("cur")
				require.NoError(t, err)
				UnsafeSetParent(parent, cur)
				return parent, cur, func() error { return parent.AddChild(cur) }
			},
			wantCyclic:           true,
			outsideGuardContract: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, cur, insert := tt.build(t)

			legacy := legacyWouldCreateCycle(parent, cur)
			got := wouldCreateCycle(parent, cur)
			require.Equal(t, tt.wantCyclic, legacy, "pre-change reference guard verdict")

			if tt.outsideGuardContract {
				require.True(t, legacy, "the reference guard must reject this shape")
				require.False(t, got, "the shipping guard's documented divergence must accept this shape")
				require.NoError(t, insert(), "the accepted insertion must go through")
				return
			}

			require.Equal(t, legacy, got, "shipping guard verdict diverges from the pre-change reference implementation")
			require.Equal(t, tt.wantCyclic, got, "shipping guard verdict")

			err := insert()
			if tt.wantCyclic {
				require.True(t, errors.Is(err, ErrCyclicNode), "expected ErrCyclicNode, got %v", err)
				return
			}
			require.NoError(t, err)
		})
	}
}

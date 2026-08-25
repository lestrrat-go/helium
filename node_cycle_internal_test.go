package helium

import (
	"errors"
	"testing"
	"time"

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
// chain, and one case built through UnsafeSetParent, whose claimant no fast-path
// source can see and which the guard therefore answers by taking the
// unconditional ancestor walk.
//
// Every case starts with unsafeParentWrites lowered, so a case that does not
// corrupt parent pointers itself really does exercise the fast exit no matter
// what an earlier test in this binary called.
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
			// The foreign-link case: the reference's only child is the shared
			// Entity node, whose parent stays the DTD. ent is therefore NOT on
			// ref's parent chain, so the ancestor walk cannot see that putting
			// ref under ent closes an ent -> ref -> ent loop; only the child-
			// pointer descent finds it, and both implementations run that same
			// descent and reject the insertion.
			//
			// The entity MUST be declared in the document's internal subset:
			// CreateReference links the Entity as the reference's child only
			// when Document.GetEntity finds the declaration, and GetEntity
			// searches intSubset/extSubset only. A detached CreateDTD joins
			// neither, which would leave ref childless and reduce this row to
			// two unrelated childless nodes. The FirstChild assertion below
			// keeps that degradation from happening silently.
			name: "ent.AddChild(ref-with-foreign-child)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				dtd, err := doc.CreateInternalSubset("root", "", "")
				require.NoError(t, err)
				ent, err := dtd.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
				require.NoError(t, err)
				ref, err := doc.CreateReference("foo")
				require.NoError(t, err)
				require.Same(t, ent, ref.FirstChild(), "the reference must carry the declared entity as its child, or this row proves nothing")
				return ent, ref, func() error { return ent.AddChild(ref) }
			},
			wantCyclic: true,
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
			// UnsafeSetParent points parent's parent pointer at cur without
			// linking parent into cur's child list, so cur stays a childless
			// *Element with no attributes: it has no claimant in any source the
			// fast exit consults. The write raises unsafeParentWrites, which puts
			// the guard back on the unconditional ancestor walk, so the walk finds
			// cur one step up and rejects the insertion exactly as the reference
			// implementation does.
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
			wantCyclic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lowerUnsafeParentWrites(t)
			parent, cur, insert := tt.build(t)

			legacy := legacyWouldCreateCycle(parent, cur)
			got := wouldCreateCycle(parent, cur)
			require.Equal(t, tt.wantCyclic, legacy, "pre-change reference guard verdict")

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

// lowerUnsafeParentWrites clears the one-way unsafeParentWrites flag for the
// duration of one test and restores it afterwards. Production code never lowers
// the flag; a test needs to, because the flag is process-wide and any earlier
// test that called UnsafeSetParent would otherwise leave every later case on the
// unconditional ancestor walk, hiding whether the fast exit still works — which
// is why BenchmarkAddChildDeepChain calls it too.
func lowerUnsafeParentWrites(t testing.TB) {
	t.Helper()
	prev := unsafeParentWrites.Swap(false)
	t.Cleanup(func() { unsafeParentWrites.Store(prev) })
}

// TestWouldCreateCycleFastExitGate pins both halves of the childless-operand
// fast exit on ONE tree: a parent pointer claiming a childless element, written
// as a raw field store so the flag's state is set by the test rather than by the
// write. With the flag lowered the guard takes the fast exit, finds no claimant
// among the operand's own sources and accepts; with the flag raised — the state
// UnsafeSetParent leaves behind — it walks parent's ancestor chain and rejects.
// An implementation that dropped the fast exit fails the first half, and one
// that ignored the flag fails the second.
func TestWouldCreateCycleFastExitGate(t *testing.T) {
	lowerUnsafeParentWrites(t)

	doc := newCycleCaseDocument()
	parent, err := doc.CreateElement("parent")
	require.NoError(t, err)
	cur, err := doc.CreateElement("cur")
	require.NoError(t, err)
	parent.baseDocNode().parent = cur

	require.True(t, legacyWouldCreateCycle(parent, cur), "the reference guard must reject this shape")
	require.False(t, wouldCreateCycle(parent, cur), "a childless operand must skip the ancestor walk while the flag is lowered")

	unsafeParentWrites.Store(true)
	require.True(t, wouldCreateCycle(parent, cur), "a raised flag must put the guard back on the ancestor walk")
}

// TestPropertiesReachTerminatesOnCyclicChain pins that the properties-chain
// search terminates on a chain knotted into a loop, and that it still finds a
// claimant that sits on the loop rather than bailing out before reaching it.
func TestPropertiesReachTerminatesOnCyclicChain(t *testing.T) {
	doc := newCycleCaseDocument()
	elem, err := doc.CreateElement("e")
	require.NoError(t, err)
	require.NoError(t, elem.SetAttribute("a", "1"))
	require.NoError(t, elem.SetAttribute("b", "2"))

	attrs := elem.Attributes()
	require.Len(t, attrs, 2)
	UnsafeSetNextSibling(attrs[1], attrs[0])

	other, err := doc.CreateElement("other")
	require.NoError(t, err)

	done := make(chan struct{})
	var reachesOther, reachesFirst, reachesLast bool
	go func() {
		defer close(done)
		reachesOther = propertiesReach(elem, other.baseDocNode())
		reachesFirst = propertiesReach(elem, attrs[0].baseDocNode())
		reachesLast = propertiesReach(elem, attrs[1].baseDocNode())
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("propertiesReach did not return: the properties walk has no termination guard")
	}

	require.False(t, reachesOther, "an unrelated node is not a claimant")
	require.True(t, reachesFirst, "the first attribute is a claimant")
	require.True(t, reachesLast, "the looping attribute is still searched before the guard stops the walk")
}

// TestAddChildCyclicPropertiesChainTerminates hands AddChild an operand whose
// attribute chain was knotted into a loop through UnsafeSetNextSibling. The
// operand is childless, so the guard resolves it through its properties chain,
// and that walk must terminate on a corrupt chain instead of spinning forever.
// The insertion itself is unrelated to the loop and must be accepted.
//
// It lowers unsafeParentWrites first: with the flag raised the guard takes its
// ancestor walk and never reads the properties chain, which would make this a
// test of nothing.
func TestAddChildCyclicPropertiesChainTerminates(t *testing.T) {
	lowerUnsafeParentWrites(t)

	doc := newCycleCaseDocument()
	e, err := doc.CreateElement("e")
	require.NoError(t, err)
	require.NoError(t, e.SetAttribute("a", "1"))
	require.NoError(t, e.SetAttribute("b", "2"))

	attrs := e.Attributes()
	require.Len(t, attrs, 2)
	// b -> a closes the properties chain into a loop.
	UnsafeSetNextSibling(attrs[1], attrs[0])

	parent, err := doc.CreateElement("parent")
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- parent.AddChild(e) }()

	select {
	case err := <-done:
		require.NoError(t, err)
		require.Same(t, Node(e), parent.FirstChild())
	case <-time.After(30 * time.Second):
		t.Fatal("AddChild did not return: the cycle guard walked a corrupt properties chain without a termination guard")
	}
}

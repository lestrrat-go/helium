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

// threeAttrElement returns a childless element carrying three attributes, plus
// its attribute list in property order. It is the starting shape for the rows
// that corrupt an attribute chain: the attributes claim the element as their
// parent, so the element's claim count is three however the chain is later cut,
// knotted, or spliced.
func threeAttrElement(t *testing.T, doc *Document) (*Element, []*Attribute) {
	t.Helper()
	e, err := doc.CreateElement("e")
	require.NoError(t, err)
	require.NoError(t, e.SetAttribute("a1", "1"))
	require.NoError(t, e.SetAttribute("a2", "2"))
	require.NoError(t, e.SetAttribute("a3", "3"))
	attrs := e.Attributes()
	require.Len(t, attrs, 3)
	return e, attrs
}

// TestWouldCreateCycleDifferential runs every adversarial case through BOTH the
// shipping guard and legacyWouldCreateCycle, the pre-change reference
// implementation, and requires their verdicts to be identical. The hand-written
// wantCyclic column is kept as a secondary assertion; it cannot establish
// equivalence on its own, because a divergence both implementations were never
// compared on is exactly what a hand-written expectation misses.
//
// The rows cover both directions the fast exit can get wrong. A row whose
// operand is claimed through a slot no enumeration can see — a severed, knotted
// or spliced properties chain, a sibling-list write, a raw unsafeSetParent —
// must still be REJECTED, because docnode.claims counts that claimant whatever
// slot does or does not hold it. A row whose operand is NOT claimed at all —
// an attribute orphaned by unsafeSetParent, an attribute moved into another
// element, an element whose attribute value merely contains a shared entity —
// must be ACCEPTED, because the ancestor walk would accept it.
// cycleDifferentialCases is the adversarial insertion table
// TestWouldCreateCycleDifferential runs. It is a function so a case can be
// replayed against another implementation of the guard without restating the
// tree it builds.
func cycleDifferentialCases() []cycleCase {
	return []cycleCase{
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
			// The attribute-pocket case: attr has no children, so the operand's
			// claim must be found through the element it hangs off rather than
			// through a child list.
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
			// itself, so the claim search must descend the attribute's value
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
			// A properties chain SEVERED behind the API's back. a2 still names e
			// as its parent, so e is still claimed and the insertion still
			// closes a loop, but no walk of e.properties can reach a2 any more.
			name: "attr.AddChild(owner with severed properties chain)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, e.SetAttribute("a1", "1"))
				require.NoError(t, e.SetAttribute("a2", "2"))
				attrs := e.Attributes()
				require.Len(t, attrs, 2)
				UnsafeSetNextSibling(attrs[0], nil)
				return attrs[1], e, func() error { return attrs[1].AddChild(e) }
			},
			wantCyclic: true,
		},
		{
			// A properties chain KNOTTED into a self-loop. The enumeration stops
			// on the loop long before it reaches a2, which still claims e.
			name: "attr.AddChild(owner with self-looping properties chain)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				e, attrs := threeAttrElement(t, doc)
				UnsafeSetNextSibling(attrs[0], attrs[0])
				return attrs[1], e, func() error { return attrs[1].AddChild(e) }
			},
			wantCyclic: true,
		},
		{
			// A SAFE RemoveAttribute over a chain whose prev pointer was
			// rewritten behind the API's back: the splice drops a3 from the
			// chain while a3.parent still names e.
			name: "attr.AddChild(owner whose safe removal dropped a claimant)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				e, attrs := threeAttrElement(t, doc)
				attrs[1].baseDocNode().prev = attrs[2]
				require.True(t, e.RemoveAttribute("a2"))
				require.Same(t, e, attrs[2].Parent(), "a3 must still name e as its parent, or this row proves nothing")
				return attrs[2], e, func() error { return attrs[2].AddChild(e) }
			},
			wantCyclic: true,
		},
		{
			// A SAFE AddSibling on an attribute that the severed chain no longer
			// reaches: the new node lands in NO slot of e — e.firstChild stays
			// nil — while naming e as its parent.
			name: "stray.AddChild(owner claimed through no slot at all)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, e.SetAttribute("a1", "1"))
				require.NoError(t, e.SetAttribute("a2", "2"))
				attrs := e.Attributes()
				require.Len(t, attrs, 2)
				UnsafeSetNextSibling(attrs[0], nil)
				stray, err := doc.CreateElement("stray")
				require.NoError(t, err)
				require.NoError(t, attrs[1].AddSibling(stray))
				require.Nil(t, e.FirstChild(), "the stray must sit in no child list, or this row proves nothing")
				require.Same(t, Node(e), stray.Parent())
				return stray, e, func() error { return stray.AddChild(e) }
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
			// unsafeSetParent points parent's parent pointer at cur without
			// linking parent into cur's child list, so cur stays a childless
			// *Element with no attributes and the claim sits in no slot at all.
			// The write still routes through setParent, so cur is claimed and
			// the guard takes the ancestor walk.
			name: "AddChild(unsafeSetParent-corrupted ancestor)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				parent, err := doc.CreateElement("parent")
				require.NoError(t, err)
				cur, err := doc.CreateElement("cur")
				require.NoError(t, err)
				unsafeSetParent(parent, cur)
				return parent, cur, func() error { return parent.AddChild(cur) }
			},
			wantCyclic: true,
		},
		{
			// A namespace-axis wrapper claims its owning element without any
			// child link, and something claims the wrapper in turn, so the
			// element really is on the insertion point's ancestor chain.
			name: "AddChild(ancestor claimed through a namespace wrapper)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				owner, err := doc.CreateElement("owner")
				require.NoError(t, err)
				parent, err := doc.CreateElement("parent")
				require.NoError(t, err)
				w := NewNamespaceNodeWrapper(NewNamespace("p", "urn:x"), owner)
				unsafeSetParent(parent, w)
				return parent, owner, func() error { return parent.AddChild(owner) }
			},
			wantCyclic: true,
		},
		{
			// An attribute ORPHANED by unsafeSetParent still sits in the
			// element's properties chain, but it no longer claims the element,
			// so the insertion closes no loop and must be accepted.
			name: "phantomAttr.AddChild(former owner)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, e.SetAttribute("a1", "1"))
				attrs := e.Attributes()
				require.Len(t, attrs, 1)
				unsafeSetParent(attrs[0], nil)
				require.Same(t, attrs[0], e.Attributes()[0], "the phantom must stay in the properties chain, or this row proves nothing")
				return attrs[0], e, func() error { return attrs[0].AddChild(e) }
			},
			wantCyclic: false,
		},
		{
			// UnsafeAppendChild moves the attribute's parent to another element
			// while leaving it in the first element's properties chain. It is no
			// longer a claimant of the first element, so the insertion is
			// accepted.
			name: "movedAttr.AddChild(former owner)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				e1, err := doc.CreateElement("e1")
				require.NoError(t, err)
				e2, err := doc.CreateElement("e2")
				require.NoError(t, err)
				require.NoError(t, e1.SetAttribute("a1", "1"))
				attrs := e1.Attributes()
				require.Len(t, attrs, 1)
				require.NoError(t, UnsafeAppendChild(e2, attrs[0]))
				require.Same(t, attrs[0], e1.Attributes()[0], "the attribute must stay in e1's properties chain, or this row proves nothing")
				return attrs[0], e1, func() error { return attrs[0].AddChild(e1) }
			},
			wantCyclic: false,
		},
		{
			// A claimant counted TWICE must not be allowed to certify that every
			// claim link was followed. The duplicate is cross-slot: attribute
			// "dup" sits in r's properties chain AND in r's child list, since
			// UnsafeAppendChild links it into the list without removing it from
			// the chain. Attribute "hidden" was reparented away, so it no longer
			// claims r, and x claims r through a pointer no slot holds. Counting
			// "dup" once per occurrence makes the enumerated total equal r's
			// claim count exactly, which would certify a search that never
			// followed x's claim link — and the loop e -> x -> r -> q -> e
			// closes unseen. Counting DISTINCT claimant identities can only
			// undercount, so the shortfall sends the guard back to the walk.
			name: "AddChild(ancestor claimed behind a double-counted attribute)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, e.SetAttribute("q", "v"))
				q := e.Attributes()[0]

				r, err := doc.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, q.AddChild(r))
				require.NoError(t, r.SetAttribute("dup", "1"))
				require.NoError(t, r.SetAttribute("hidden", "2"))
				require.NoError(t, r.SetAttribute("tail", "3"))
				rattrs := r.Attributes()
				require.Len(t, rattrs, 3)

				// dup now occupies both of r's enumerable slots at once.
				require.NoError(t, UnsafeAppendChild(r, rattrs[0]))
				require.Same(t, rattrs[0], r.FirstChild(), "dup must be in r's child list, or this row proves nothing")
				require.Same(t, rattrs[0], r.Attributes()[0], "dup must stay in r's properties chain, or this row proves nothing")

				// hidden stops claiming r, freeing the slot dup double-counts.
				elsewhere, err := doc.CreateElement("elsewhere")
				require.NoError(t, err)
				unsafeSetParent(rattrs[1], elsewhere)

				// x claims r through a pointer no enumeration can reach.
				x, err := doc.CreateElement("x")
				require.NoError(t, err)
				unsafeSetParent(x, r)
				require.EqualValues(t, 3, r.baseDocNode().claims, "r must be claimed by dup, tail and x, or this row proves nothing")

				require.Nil(t, e.FirstChild(), "e must stay childless, or the guard never reaches the claim search")
				return x, e, func() error { return x.AddChild(e) }
			},
			wantCyclic: true,
		},
		{
			// No Unsafe call anywhere: an element whose attribute value expands a
			// declared entity shares that entity's node with the DTD. The shared
			// entity is reachable from the element by CHILD pointers, but it is
			// not on the insertion point's ancestor chain, so adding the element
			// as a sibling of the entity's own text closes no loop.
			name: "entityText.AddSibling(element whose attribute expands the entity)",
			build: func(t *testing.T) (Node, Node, func() error) {
				doc := newCycleCaseDocument()
				dtd, err := doc.CreateInternalSubset("root", "", "")
				require.NoError(t, err)
				ent, err := dtd.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
				require.NoError(t, err)
				e, err := doc.CreateElement("e")
				require.NoError(t, err)
				require.NoError(t, e.SetParsedAttribute("a", "&foo;"))
				inner := ent.FirstChild()
				require.NotNil(t, inner, "the entity must carry its expansion, or this row proves nothing")
				m, ok := inner.(MutableNode)
				require.True(t, ok)
				return inner.Parent(), e, func() error { return m.AddSibling(e) }
			},
			wantCyclic: false,
		},
	}
}

func TestWouldCreateCycleDifferential(t *testing.T) {
	for _, tt := range cycleDifferentialCases() {
		t.Run(tt.name, func(t *testing.T) {
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

// TestWouldCreateCycleFastExitGate pins both halves of the childless-operand
// fast exit on ONE tree. With nothing claiming cur, the guard exits on the claim
// count alone and accepts, even though parent's own pointer is about to be
// overwritten; once unsafeSetParent points parent at cur, cur is claimed and the
// guard takes the ancestor walk and rejects. Clearing the pointer again drops
// the claim and restores the fast exit, which the process-wide flag this
// replaced could never do.
func TestWouldCreateCycleFastExitGate(t *testing.T) {
	doc := newCycleCaseDocument()
	parent, err := doc.CreateElement("parent")
	require.NoError(t, err)
	cur, err := doc.CreateElement("cur")
	require.NoError(t, err)

	require.Zero(t, cur.baseDocNode().claims)
	require.False(t, wouldCreateCycle(parent, cur), "an unclaimed operand must skip the ancestor walk")

	unsafeSetParent(parent, cur)
	require.EqualValues(t, 1, cur.baseDocNode().claims)
	require.True(t, legacyWouldCreateCycle(parent, cur), "the reference guard must reject this shape")
	require.True(t, wouldCreateCycle(parent, cur), "a claimed operand must go back on the ancestor walk")

	unsafeSetParent(parent, nil)
	require.Zero(t, cur.baseDocNode().claims)
	require.False(t, wouldCreateCycle(parent, cur), "dropping the claim must restore the fast exit")
}

// TestClaimsReachTerminatesOnCyclicChain pins that the claim search terminates
// on a properties chain knotted into a loop, and that it reports the answer as
// INEXACT there: the loop stops the enumeration short of the element's claim
// count, so the caller must fall back to the ancestor walk instead of trusting a
// partial search.
func TestClaimsReachTerminatesOnCyclicChain(t *testing.T) {
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
	var otherExact, firstFound, lastFound bool
	go func() {
		defer close(done)
		_, otherExact = claimsReach(elem, other.baseDocNode())
		firstFound, _ = claimsReach(elem, attrs[0].baseDocNode())
		lastFound, _ = claimsReach(elem, attrs[1].baseDocNode())
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("claimsReach did not return: the claim search has no termination guard")
	}

	require.False(t, otherExact, "a knotted chain cannot be enumerated in full, so the answer is inexact")
	require.True(t, firstFound, "the first attribute is a claimant")
	require.True(t, lastFound, "the looping attribute is still searched before the guard stops the walk")
}

// TestAddChildCyclicPropertiesChainTerminates hands AddChild an operand whose
// attribute chain was knotted into a loop through UnsafeSetNextSibling. The
// operand is childless and claimed, so the guard reads that chain, and the read
// must terminate on a corrupt chain instead of spinning forever. The insertion
// itself is unrelated to the loop and must be accepted.
func TestAddChildCyclicPropertiesChainTerminates(t *testing.T) {
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

// collectClaimNodes gathers every node reachable from n through the slots that
// can hold a parent-bearing link: the child list, an Element's properties chain,
// and a Document's DTD subsets. It is the test-side enumeration used to
// recompute docnode.claims from scratch.
func collectClaimNodes(n Node, out map[*docnode]Node) {
	dn := n.baseDocNode()
	if _, seen := out[dn]; seen {
		return
	}
	out[dn] = n
	if elem, ok := n.(*Element); ok {
		for attr := elem.properties; attr != nil; attr = attr.NextAttribute() {
			collectClaimNodes(attr, out)
		}
	}
	if doc, ok := n.(*Document); ok {
		if doc.intSubset != nil {
			collectClaimNodes(doc.intSubset, out)
		}
		if doc.extSubset != nil {
			collectClaimNodes(doc.extSubset, out)
		}
	}
	for child := dn.firstChild; child != nil; child = child.baseDocNode().next {
		collectClaimNodes(child, out)
	}
}

// requireClaimsExact recomputes every reachable node's claim count from the
// parent pointers actually present and requires docnode.claims to match. This is
// the invariant the whole fast exit rests on: setParent is the sole writer of
// docnode.parent, so claims is the exact number of nodes naming a given node as
// their parent. Any mutation path that assigned the field directly would show up
// here as a count that drifted.
func requireClaimsExact(t *testing.T, roots ...Node) {
	t.Helper()
	nodes := map[*docnode]Node{}
	for _, r := range roots {
		collectClaimNodes(r, nodes)
	}
	want := map[*docnode]int32{}
	for dn := range nodes {
		if dn.parent == nil {
			continue
		}
		want[dn.parent.baseDocNode()]++
	}
	for dn, n := range nodes {
		require.Equal(t, want[dn], dn.claims,
			"claim count drifted on %s node %q", n.Type(), dn.name)
	}
}

// TestClaimsStayExactThroughMutationAPI drives the whole guarded mutation
// surface — attribute create/replace/remove, child append, sibling splice,
// replace, unlink, deep copy, document element install — and recomputes every
// claim count from the parent pointers afterwards. It is the chokepoint's
// contract test: a parent-field write that skipped setParent leaves a count
// behind here.
func TestClaimsStayExactThroughMutationAPI(t *testing.T) {
	doc := newCycleCaseDocument()
	_, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))

	kid, err := doc.CreateElement("kid")
	require.NoError(t, err)
	require.NoError(t, kid.SetAttribute("a", "1"))
	require.NoError(t, kid.SetAttributeNS("b", "2", NewNamespace("p", "urn:x")))
	require.NoError(t, root.AddChild(kid))
	requireClaimsExact(t, doc)

	// Replacing an attribute in place detaches the old one.
	require.NoError(t, kid.SetAttribute("a", "1-updated"))
	require.True(t, kid.RemoveAttribute("a"))
	requireClaimsExact(t, doc)

	sib, err := doc.CreateElement("sib")
	require.NoError(t, err)
	require.NoError(t, kid.AddSibling(sib))
	requireClaimsExact(t, doc)

	repl, err := doc.CreateElement("repl")
	require.NoError(t, err)
	require.NoError(t, sib.Replace(repl))
	requireClaimsExact(t, doc, sib)
	require.Zero(t, sib.baseDocNode().claims)

	copied, err := CopyNode(kid, doc)
	require.NoError(t, err)
	require.NoError(t, root.AddChild(copied))
	requireClaimsExact(t, doc)

	UnlinkNode(kid)
	requireClaimsExact(t, doc, kid)

	// The unlinked subtree keeps its own internal claims and none on the tree.
	require.Nil(t, kid.Parent())
	requireClaimsExact(t, kid)
}

// TestClaimsReturnToZero pins that a node built up and then torn back down
// through the public API ends with no claimants at all, so the fast exit is
// available again. A counter that only ever incremented would fail here, and
// with it the whole point of replacing the one-way flag this chokepoint retired.
func TestClaimsReturnToZero(t *testing.T) {
	doc := newCycleCaseDocument()
	e, err := doc.CreateElement("e")
	require.NoError(t, err)
	require.NoError(t, e.SetAttribute("a", "1"))
	require.NoError(t, e.SetAttribute("b", "2"))
	child, err := doc.CreateElement("child")
	require.NoError(t, err)
	require.NoError(t, e.AddChild(child))
	require.EqualValues(t, 3, e.baseDocNode().claims)

	require.True(t, e.RemoveAttribute("a"))
	require.True(t, e.RemoveAttribute("b"))
	UnlinkNode(child)
	require.Zero(t, e.baseDocNode().claims)
}

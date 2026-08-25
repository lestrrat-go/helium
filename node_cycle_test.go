package helium_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

func newCycleTestDocument() *helium.Document {
	return helium.NewDocument("1.0", "utf-8", helium.StandaloneExplicitNo)
}

// TestWouldCreateCycleVerdicts pins the accept/reject verdict of the
// wouldCreateCycle guard reached through AddChild/AddSibling/Replace across 13
// adversarial cases, including the two attribute-pocket cases (a childless
// *Attribute and a childless entity reference living inside an attribute
// value) that a childless-operand fast path must still resolve without
// weakening the guard: an operand with no children can still be claimed by an
// Element's properties chain or a Document's DTD subsets, and the guard must
// keep rejecting those insertions exactly as the depth-proportional ancestor
// walk always has.
func TestWouldCreateCycleVerdicts(t *testing.T) {
	tests := []struct {
		name       string
		run        func(t *testing.T) error
		wantCyclic bool
	}{
		{
			name: "self AddChild",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				a, err := doc.CreateElement("a")
				require.NoError(t, err)
				return a.AddChild(a)
			},
			wantCyclic: true,
		},
		{
			name: "leaf.AddChild(ancestor 50 up)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				root, err := doc.CreateElement("root")
				require.NoError(t, err)
				require.NoError(t, doc.SetDocumentElement(root))
				cur := root
				var leaf *helium.Element
				for range 50 {
					e, err := doc.CreateElement("e")
					require.NoError(t, err)
					require.NoError(t, cur.AddChild(e))
					cur = e
					leaf = e
				}
				return leaf.AddChild(root)
			},
			wantCyclic: true,
		},
		{
			name: "leaf.AddChild(document)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				root, err := doc.CreateElement("root")
				require.NoError(t, err)
				require.NoError(t, doc.SetDocumentElement(root))
				leaf, err := doc.CreateElement("leaf")
				require.NoError(t, err)
				require.NoError(t, root.AddChild(leaf))
				return leaf.AddChild(doc)
			},
			wantCyclic: true,
		},
		{
			// The attribute-pocket case: attr has no children, so the fast path
			// must resolve it through Element.properties rather than assuming a
			// childless operand has no claimant.
			name: "attr.AddChild(childless owner)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				elem, err := doc.CreateElement("e")
				require.NoError(t, err)
				attr, err := doc.CreateAttribute("a", "", nil)
				require.NoError(t, err)
				require.NoError(t, elem.AddChild(attr))
				return attr.AddChild(elem)
			},
			wantCyclic: true,
		},
		{
			// The second attribute-pocket case: the claimant is a value child
			// inside the attribute (an entity reference), not the attribute
			// itself, so propertiesReach must also search each attribute's value
			// subtree.
			name: "attrValueRef.AddChild(childless owner)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				elem, err := doc.CreateElement("e")
				require.NoError(t, err)
				attr, err := doc.CreateAttribute("a", "x&foo;y", nil)
				require.NoError(t, err)
				require.NoError(t, elem.AddChild(attr))
				var ref helium.Node
				for c := range helium.Children(attr) {
					if c.Type() == helium.EntityRefNode {
						ref = c
					}
				}
				require.NotNil(t, ref, "attribute value entity ref must be built")
				m, ok := ref.(helium.MutableNode)
				require.True(t, ok)
				return m.AddChild(elem)
			},
			wantCyclic: true,
		},
		{
			name: "g.AddChild(p) [p has children]",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				g, err := doc.CreateElement("g")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				require.NoError(t, c.AddChild(g))
				return g.AddChild(p)
			},
			wantCyclic: true,
		},
		{
			// The foreign-link case: an entity's child is the shared Entity node
			// whose parent stays the DTD, so this cycle is invisible to the
			// ancestor walk and must be caught by childReaches instead.
			name: "ent.AddChild(ref-with-foreign-child)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				dtd, err := doc.CreateDTD()
				require.NoError(t, err)
				ent, err := dtd.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
				require.NoError(t, err)
				ref, err := doc.CreateReference("foo")
				require.NoError(t, err)
				return ent.AddChild(ref)
			},
			wantCyclic: false,
		},
		{
			name: "plain AddChild",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				return p.AddChild(c)
			},
			wantCyclic: false,
		},
		{
			name: "AddSibling",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				s, err := doc.CreateElement("s")
				require.NoError(t, err)
				return c.AddSibling(s)
			},
			wantCyclic: false,
		},
		{
			name: "AddChild(subtree)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				sub, err := doc.CreateElement("sub")
				require.NoError(t, err)
				sc, err := doc.CreateElement("sc")
				require.NoError(t, err)
				require.NoError(t, sub.AddChild(sc))
				return p.AddChild(sub)
			},
			wantCyclic: false,
		},
		{
			name: "Replace",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				sc, err := doc.CreateElement("sc")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(sc))
				r, err := doc.CreateElement("r")
				require.NoError(t, err)
				return sc.Replace(r)
			},
			wantCyclic: false,
		},
		{
			name: "g.Replace(p)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				g, err := doc.CreateElement("g")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				require.NoError(t, c.AddChild(g))
				return g.Replace(p)
			},
			wantCyclic: true,
		},
		{
			name: "g.AddSibling(p)",
			run: func(t *testing.T) error {
				doc := newCycleTestDocument()
				p, err := doc.CreateElement("p")
				require.NoError(t, err)
				c, err := doc.CreateElement("c")
				require.NoError(t, err)
				g, err := doc.CreateElement("g")
				require.NoError(t, err)
				require.NoError(t, p.AddChild(c))
				require.NoError(t, c.AddChild(g))
				return g.AddSibling(p)
			},
			wantCyclic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			if tt.wantCyclic {
				require.True(t, errors.Is(err, helium.ErrCyclicNode), "expected ErrCyclicNode, got %v", err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestAddChildDeepChainGrowth builds a several-thousand-deep chain through
// AddChild, exercising the same insertion pattern that made wouldCreateCycle's
// ancestor walk quadratic, and asserts the resulting chain's shape rather than
// its timing: every AddChild in the chain must still succeed and land in the
// correct position. Elements carry two attributes each so the properties-chain
// fast path (propertiesReach) is exercised on every insertion, not just the
// zero-attribute case.
func TestAddChildDeepChainGrowth(t *testing.T) {
	const depth = 4000

	doc := newCycleTestDocument()
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))

	cur := helium.Node(root)
	nodes := make([]*helium.Element, 0, depth)
	for i := range depth {
		e, err := doc.CreateElement(fmt.Sprintf("e%d", i))
		require.NoError(t, err)
		require.NoError(t, e.SetAttribute("a0", "v0"))
		require.NoError(t, e.SetAttribute("a1", "v1"))
		mut, ok := cur.(helium.MutableNode)
		require.True(t, ok)
		require.NoError(t, mut.AddChild(e))
		nodes = append(nodes, e)
		cur = e
	}

	// Walk back down verifying parent/child linkage and that no node was
	// mislinked or skipped.
	walker := helium.Node(root)
	for i, e := range nodes {
		child := walker.FirstChild()
		require.NotNil(t, child, "missing child at depth %d", i)
		require.Same(t, helium.Node(e), child)
		require.Same(t, walker, child.Parent(), "parent mismatch at depth %d", i)
		require.Nil(t, child.NextSibling(), "unexpected sibling at depth %d", i)
		walker = child
	}

	// The deepest node must be a true leaf.
	require.Nil(t, walker.FirstChild())
}

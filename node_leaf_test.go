package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// leafCtor builds a freshly created, unlinked leaf node of a specific concrete
// type for the table-driven AddChild/AddSibling guard tests.
type leafCtor func(t *testing.T, doc *helium.Document) helium.MutableNode

func mustCreatePI(t *testing.T, doc *helium.Document) *helium.ProcessingInstruction {
	t.Helper()
	return doc.CreatePI("target", "data")
}

func mustCreateEntityRef(t *testing.T, doc *helium.Document) *helium.EntityRef {
	t.Helper()
	ref, err := doc.CreateReference("amp")
	require.NoError(t, err, "CreateReference must succeed")
	return ref
}

// leafCase describes one concrete leaf type and the guard behavior it exposes
// through its AddChild override.
type leafCase struct {
	name string
	// new builds a fresh, unlinked instance of the leaf type.
	new leafCtor
	// canContainChildren is true when the type's AddChild can legally accept a
	// foreign child (EntityRef). For those the ancestor-insertion guard is
	// exercised. Types whose AddChild only ever rejects (CDATASection, PI) or
	// only content-merges its own kind (Text, Comment) set this false.
	canContainChildren bool
	// addChildSelfErr is the exact error AddChild(self) must return. For Text,
	// Comment and EntityRef the shared cycle guard fires; CDATASection and PI
	// reject AddChild before reaching the shared cycle guard (CDATASection with
	// ErrInvalidOperation, PI with errPIAddChild for a non-text node), so
	// self-insertion surfaces that rejection instead.
	addChildSelfErr string
}

func leafCases() []leafCase {
	return []leafCase{
		{
			name: "Text",
			new: func(t *testing.T, doc *helium.Document) helium.MutableNode {
				return mustCreateText(t, doc, []byte("x"))
			},
			addChildSelfErr: errAddChildCycle,
		},
		{
			name: "Comment",
			new: func(t *testing.T, doc *helium.Document) helium.MutableNode {
				return mustCreateComment(t, doc, []byte("x"))
			},
			addChildSelfErr: errAddChildCycle,
		},
		{
			name: "CDATASection",
			new: func(t *testing.T, doc *helium.Document) helium.MutableNode {
				return doc.CreateCDATASection([]byte("x"))
			},
			addChildSelfErr: helium.ErrInvalidOperation.Error(),
		},
		{
			// A PI carries its content as a string, not as element/text
			// children, so its AddChild rejects a foreign (non-text) node
			// before the shared cycle guard, just like CDATASection. A PI is
			// not a text node, so PI.AddChild(self) hits that rejection.
			name:            "ProcessingInstruction",
			new:             func(t *testing.T, doc *helium.Document) helium.MutableNode { return mustCreatePI(t, doc) },
			addChildSelfErr: errPIAddChild,
		},
		{
			name:               "EntityRef",
			new:                func(t *testing.T, doc *helium.Document) helium.MutableNode { return mustCreateEntityRef(t, doc) },
			canContainChildren: true,
			addChildSelfErr:    errAddChildCycle,
		},
	}
}

const (
	errAddChildCycle   = "cannot add a node as a child of itself or one of its descendants"
	errAddSiblingCycle = "cannot add a node as a sibling of itself or one of its descendants"
	errPIAddChild      = "helium: cannot add ProcessingInstructionNode as a child of a processing instruction"
)

// TestLeafAddChildGuards exercises every leaf-type AddChild override against the
// shared self/ancestor cycle guard and auto-unlink contract.
func TestLeafAddChildGuards(t *testing.T) {
	t.Parallel()

	for _, tc := range leafCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("self insertion is rejected", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				leaf := tc.new(t, doc)
				require.NoError(t, root.AddChild(leaf), "leaf starts under root")

				err := leaf.AddChild(leaf)
				require.Error(t, err, "AddChild(self) must be rejected")
				require.ErrorContains(t, err, tc.addChildSelfErr)

				require.Nil(t, leaf.FirstChild(), "leaf must not gain a child")
				require.Equal(t, helium.Node(leaf), root.FirstChild(), "tree must not be corrupted")
				require.Equal(t, helium.Node(root), leaf.Parent(), "leaf parent must stay root")
				requireNoCycle(t, root)
			})

			if tc.canContainChildren {
				t.Run("ancestor insertion is rejected", func(t *testing.T) {
					t.Parallel()
					doc := helium.NewDefaultDocument()
					root := mustCreateElement(t, doc, "root")
					leaf := tc.new(t, doc)
					require.NoError(t, root.AddChild(leaf), "leaf starts under root")

					// Inserting root (an ancestor of leaf) as a child of leaf would
					// make an ancestor a descendant of itself.
					err := leaf.AddChild(root)
					require.Error(t, err, "inserting an ancestor must be rejected")
					require.ErrorContains(t, err, errAddChildCycle)

					require.Nil(t, leaf.FirstChild(), "leaf must not gain a child")
					require.Nil(t, root.Parent(), "root must remain the tree root")
					require.Equal(t, helium.Node(root), leaf.Parent(), "tree must stay intact")
					requireNoCycle(t, root)
				})

				t.Run("legal already-linked move unlinks from old parent", func(t *testing.T) {
					t.Parallel()
					doc := helium.NewDefaultDocument()
					container := tc.new(t, doc)
					oldParent := mustCreateElement(t, doc, "old")
					moving := mustCreateElement(t, doc, "moving")
					require.NoError(t, oldParent.AddChild(moving), "moving starts under old parent")

					// Moving moving from oldParent into the leaf container must detach
					// it from oldParent first, leaving no stale links behind.
					require.NoError(t, container.AddChild(moving), "reparenting into leaf container succeeds")

					require.Equal(t, helium.Node(container), moving.Parent(), "moving parent is now the container")
					require.Equal(t, helium.Node(moving), container.FirstChild(), "container firstChild is moving")
					require.Equal(t, helium.Node(moving), container.LastChild(), "container lastChild is moving")
					require.Nil(t, oldParent.FirstChild(), "old parent no longer holds moving")
					require.Nil(t, oldParent.LastChild(), "old parent no longer holds moving")
					require.Nil(t, moving.PrevSibling(), "moving has no stale prev")
					require.Nil(t, moving.NextSibling(), "moving has no stale next")
					requireNoCycle(t, container)
					requireNoCycle(t, oldParent)
				})
			}
		})
	}
}

// TestLeafAddSiblingGuards exercises every leaf-type AddSibling override against
// the shared self/ancestor cycle guard and auto-unlink contract.
func TestLeafAddSiblingGuards(t *testing.T) {
	t.Parallel()

	for _, tc := range leafCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("self insertion is rejected", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				leaf := tc.new(t, doc)
				require.NoError(t, root.AddChild(leaf), "leaf starts under root")

				err := leaf.AddSibling(leaf)
				require.Error(t, err, "AddSibling(self) must be rejected")
				require.ErrorContains(t, err, errAddSiblingCycle)

				require.Equal(t, helium.Node(leaf), root.FirstChild(), "tree must not be corrupted")
				require.Equal(t, helium.Node(leaf), root.LastChild(), "tree must not be corrupted")
				require.Nil(t, leaf.NextSibling(), "leaf must not gain a sibling")
				require.Nil(t, leaf.PrevSibling(), "leaf must not gain a sibling")
				requireNoCycle(t, root)
			})

			t.Run("ancestor insertion is rejected", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				mid := mustCreateElement(t, doc, "mid")
				leaf := tc.new(t, doc)
				require.NoError(t, root.AddChild(mid), "mid starts under root")
				require.NoError(t, mid.AddChild(leaf), "leaf starts under mid")

				// A sibling of leaf lands under mid; installing root (mid's ancestor)
				// there would make an ancestor a descendant of itself.
				err := leaf.AddSibling(root)
				require.Error(t, err, "inserting an ancestor as a sibling must be rejected")
				require.ErrorContains(t, err, errAddSiblingCycle)

				require.Nil(t, root.Parent(), "root must remain the tree root")
				require.Equal(t, helium.Node(leaf), mid.FirstChild(), "tree must stay intact")
				require.Equal(t, helium.Node(leaf), mid.LastChild(), "tree must stay intact")
				require.Nil(t, leaf.NextSibling(), "leaf must not gain a sibling")
				requireNoCycle(t, root)
			})

			t.Run("legal already-linked move unlinks from old parent", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				dst := mustCreateElement(t, doc, "dst")
				src := mustCreateElement(t, doc, "src")
				anchor := tc.new(t, doc)
				moving := mustCreateElement(t, doc, "moving")
				require.NoError(t, dst.AddChild(anchor), "anchor starts under dst")
				require.NoError(t, src.AddChild(moving), "moving starts under src")

				// anchor.AddSibling(moving) must detach moving from src before
				// splicing it after anchor under dst.
				require.NoError(t, anchor.AddSibling(moving), "moving anchor's sibling succeeds")

				require.Equal(t, helium.Node(dst), moving.Parent(), "moving parent is now dst")
				require.Equal(t, helium.Node(moving), anchor.NextSibling(), "moving follows anchor")
				require.Equal(t, helium.Node(anchor), moving.PrevSibling(), "anchor precedes moving")
				require.Equal(t, helium.Node(moving), dst.LastChild(), "dst lastChild is moving")
				require.Nil(t, src.FirstChild(), "src no longer holds moving")
				require.Nil(t, src.LastChild(), "src no longer holds moving")
				requireNoCycle(t, dst)
				requireNoCycle(t, src)
			})
		})
	}
}

const errReplaceCycle = "cannot replace a node with one of its own ancestors"

// TestLeafReplaceGuards exercises every leaf-type Replace override (Text,
// Comment, CDATASection, ProcessingInstruction, EntityRef). Each override
// delegates to replaceNode, so this confirms the override exists and is wired to
// the shared ancestor-rejection guard and to the linked / non-MutableNode
// replacement splice.
func TestLeafReplaceGuards(t *testing.T) {
	t.Parallel()

	for _, tc := range leafCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("ancestor replacement is rejected", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				mid := mustCreateElement(t, doc, "mid")
				leaf := tc.new(t, doc)
				require.NoError(t, root.AddChild(mid), "mid starts under root")
				require.NoError(t, mid.AddChild(leaf), "leaf starts under mid")

				// Replacing leaf with mid (its parent/ancestor) would splice an
				// ancestor below itself, creating a cycle.
				err := leaf.Replace(mid)
				require.Error(t, err, "replacing leaf with an ancestor must be rejected")
				require.ErrorContains(t, err, errReplaceCycle)

				require.Equal(t, helium.Node(leaf), mid.FirstChild(), "leaf stays under mid")
				require.Equal(t, helium.Node(mid), leaf.Parent(), "leaf parent stays mid")
				requireNoCycle(t, root)
			})

			t.Run("legal already-linked replacement splices in", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				leaf := tc.new(t, doc)
				incoming := mustCreateElement(t, doc, "incoming")
				require.NoError(t, root.AddChild(leaf), "leaf starts under root")

				src := mustCreateElement(t, doc, "src")
				require.NoError(t, src.AddChild(incoming), "incoming starts under src")

				// incoming is already linked under src; replacing leaf with it must
				// detach it from src and splice it into leaf's position under root.
				require.NoError(t, leaf.Replace(incoming), "linked replacement succeeds")

				require.Equal(t, helium.Node(incoming), root.FirstChild(), "incoming took leaf's position")
				require.Equal(t, helium.Node(incoming), root.LastChild(), "incoming is the only child")
				require.Equal(t, helium.Node(root), incoming.Parent(), "incoming parent is root")
				require.Nil(t, src.FirstChild(), "src no longer holds incoming")
				require.Nil(t, leaf.Parent(), "replaced leaf is detached")
				require.Nil(t, leaf.NextSibling(), "replaced leaf has no stale next")
				require.Nil(t, leaf.PrevSibling(), "replaced leaf has no stale prev")
				requireNoCycle(t, root)
				requireNoCycle(t, src)
			})

			t.Run("non-mutable replacement splices in", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				leaf := tc.new(t, doc)
				require.NoError(t, root.AddChild(leaf), "leaf starts under root")

				ns := helium.NewNamespace("p", "urn:example")
				nsw := helium.NewNamespaceNodeWrapper(ns, nil)

				// A non-MutableNode replacement must not panic on a force-cast and
				// must take leaf's position under root.
				require.NoError(t, leaf.Replace(nsw), "non-mutable replacement succeeds")

				require.Equal(t, helium.Node(nsw), root.FirstChild(), "wrapper took leaf's position")
				require.Equal(t, helium.Node(root), nsw.Parent(), "wrapper parent is root")
				require.Nil(t, leaf.Parent(), "replaced leaf is detached")
				requireNoCycle(t, root)
			})
		})
	}
}

// TestLeafFastPathNilOperand verifies that the leaf-node AddChild/AddSibling
// overrides which run a content-merge fast path (Text.AddChild, Text.AddSibling,
// Comment.AddChild, ProcessingInstruction.AddChild) reject a nil operand with
// ErrNilNode instead of panicking,
// and leave the linked leaf untouched. Both a literal nil interface and a
// matching typed-nil concrete pointer (Go's interface nil trap) are exercised,
// since the overrides run a type assertion / debug log / preflight before the
// guard would otherwise be reached.
func TestLeafFastPathNilOperand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// op runs the override under test against the supplied operand.
		op func(leaf helium.MutableNode, cur helium.Node) error
		// typedNil returns a matching typed-nil operand wrapped in a non-nil Node
		// interface, to exercise the same fast path's interface-nil trap.
		typedNil func() helium.Node
		// newLeaf builds the linked leaf that receives the call.
		newLeaf leafCtor
	}{
		{
			name: "Text.AddChild",
			op:   func(leaf helium.MutableNode, cur helium.Node) error { return leaf.AddChild(cur) },
			typedNil: func() helium.Node {
				var tn *helium.Text
				return tn
			},
			newLeaf: func(t *testing.T, doc *helium.Document) helium.MutableNode {
				return mustCreateText(t, doc, []byte("x"))
			},
		},
		{
			name: "Text.AddSibling",
			op:   func(leaf helium.MutableNode, cur helium.Node) error { return leaf.AddSibling(cur) },
			typedNil: func() helium.Node {
				var tn *helium.Text
				return tn
			},
			newLeaf: func(t *testing.T, doc *helium.Document) helium.MutableNode {
				return mustCreateText(t, doc, []byte("x"))
			},
		},
		{
			name: "Comment.AddChild",
			op:   func(leaf helium.MutableNode, cur helium.Node) error { return leaf.AddChild(cur) },
			typedNil: func() helium.Node {
				var cn *helium.Comment
				return cn
			},
			newLeaf: func(t *testing.T, doc *helium.Document) helium.MutableNode {
				return mustCreateComment(t, doc, []byte("x"))
			},
		},
		{
			name: "PI.AddChild",
			op:   func(leaf helium.MutableNode, cur helium.Node) error { return leaf.AddChild(cur) },
			typedNil: func() helium.Node {
				// A typed-nil *Text would otherwise reach the Text/CDATA
				// content-merge fast path and panic on cur.Type().
				var tn *helium.Text
				return tn
			},
			newLeaf: func(t *testing.T, doc *helium.Document) helium.MutableNode {
				return mustCreatePI(t, doc)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("literal nil", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				leaf := tc.newLeaf(t, doc)
				require.NoError(t, root.AddChild(leaf), "leaf starts under root")

				err := tc.op(leaf, nil)
				require.ErrorIs(t, err, helium.ErrNilNode, "literal nil operand must return ErrNilNode")

				require.Nil(t, leaf.FirstChild(), "leaf must not gain a child")
				require.Nil(t, leaf.NextSibling(), "leaf must not gain a sibling")
				require.Equal(t, helium.Node(leaf), root.FirstChild(), "tree must not be corrupted")
				require.Equal(t, helium.Node(root), leaf.Parent(), "leaf parent must stay root")
				requireNoCycle(t, root)
			})

			t.Run("typed nil", func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root := mustCreateElement(t, doc, "root")
				leaf := tc.newLeaf(t, doc)
				require.NoError(t, root.AddChild(leaf), "leaf starts under root")

				err := tc.op(leaf, tc.typedNil())
				require.ErrorIs(t, err, helium.ErrNilNode, "typed-nil operand must return ErrNilNode")

				require.Nil(t, leaf.FirstChild(), "leaf must not gain a child")
				require.Nil(t, leaf.NextSibling(), "leaf must not gain a sibling")
				require.Equal(t, helium.Node(leaf), root.FirstChild(), "tree must not be corrupted")
				require.Equal(t, helium.Node(root), leaf.Parent(), "leaf parent must stay root")
				requireNoCycle(t, root)
			})
		})
	}
}

// TestTextAddSiblingNonTextFallback covers Text.AddSibling's non-text fallback
// path (the `return addSibling(n, cur)` branch). Moving an already-linked
// non-text node via text.AddSibling must auto-unlink it from its old parent and
// fix the sibling/parent pointers.
func TestTextAddSiblingNonTextFallback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		newNode func(t *testing.T, doc *helium.Document) helium.Node
	}{
		{
			name:    "comment incoming",
			newNode: func(t *testing.T, doc *helium.Document) helium.Node { return mustCreateComment(t, doc, []byte("note")) },
		},
		{
			name:    "element incoming",
			newNode: func(t *testing.T, doc *helium.Document) helium.Node { return mustCreateElement(t, doc, "incoming") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			dst := mustCreateElement(t, doc, "dst")
			src := mustCreateElement(t, doc, "src")
			txt := mustCreateText(t, doc, []byte("anchor"))
			incoming := tc.newNode(t, doc)

			require.NoError(t, dst.AddChild(txt), "text anchor starts under dst")
			require.NoError(t, src.AddChild(incoming), "incoming starts under src")

			// incoming is non-text, so Text.AddSibling falls through to the generic
			// addSibling path, which must auto-unlink it from src first.
			require.NoError(t, txt.AddSibling(incoming), "non-text sibling move succeeds")

			require.Equal(t, helium.Node(dst), incoming.Parent(), "incoming parent is now dst")
			require.Equal(t, incoming, txt.NextSibling(), "incoming follows the text node")
			require.Equal(t, helium.Node(txt), incoming.PrevSibling(), "text node precedes incoming")
			require.Equal(t, incoming, dst.LastChild(), "dst lastChild is incoming")
			require.Nil(t, src.FirstChild(), "src no longer holds incoming")
			require.Nil(t, src.LastChild(), "src no longer holds incoming")
			require.Nil(t, incoming.NextSibling(), "incoming has no stale next")
			requireNoCycle(t, dst)
			requireNoCycle(t, src)
		})
	}
}

// TestPIContentIsStringNotChildren verifies that a processing instruction
// carries its content as a string (the "data" portion), not as child nodes.
// AppendText and a Text/CDATA AddChild route into the PI data; any other node
// type is rejected. A PI must never gain element/text children, and must
// serialize as "<?target data?>".
func TestPIContentIsStringNotChildren(t *testing.T) {
	t.Parallel()

	t.Run("AppendText routes into data and creates no children", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		pi := doc.CreatePI("target", "")

		require.NoError(t, pi.AppendText([]byte("hello")), "AppendText must succeed")
		require.NoError(t, pi.AppendText([]byte(" world")), "second AppendText must succeed")

		require.Nil(t, pi.FirstChild(), "PI must not gain child nodes")
		require.Nil(t, pi.LastChild(), "PI must not gain child nodes")
		require.Equal(t, "hello world", string(pi.Content()), "PI data must hold the appended text")
	})

	t.Run("AddChild of text/cdata routes into data and creates no children", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		pi := doc.CreatePI("target", "a")

		require.NoError(t, pi.AddChild(doc.CreateText([]byte("b"))), "Text AddChild must succeed")
		require.NoError(t, pi.AddChild(doc.CreateCDATASection([]byte("c"))), "CDATA AddChild must succeed")

		require.Nil(t, pi.FirstChild(), "PI must not gain child nodes")
		require.Equal(t, "abc", string(pi.Content()), "PI data must absorb text/cdata content")
	})

	t.Run("AddChild unlinks an already-linked text/cdata operand", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		oldParent := mustCreateElement(t, doc, "old")
		txt := doc.CreateText([]byte("b"))
		require.NoError(t, oldParent.AddChild(txt), "text starts under oldParent")
		require.Equal(t, helium.Node(oldParent), txt.Parent(), "text parent is oldParent")

		pi := doc.CreatePI("target", "a")
		require.NoError(t, pi.AddChild(txt), "Text AddChild must succeed")

		// Content is merged into the PI data...
		require.Equal(t, "ab", string(pi.Content()), "PI data must absorb the text content")
		require.Nil(t, pi.FirstChild(), "PI must not gain a child")
		// ...and the source node is unlinked from its old parent, honoring the
		// AddChild auto-unlink contract instead of leaving it linked twice.
		require.Nil(t, txt.Parent(), "merged text must be unlinked from its old parent")
		require.Nil(t, oldParent.FirstChild(), "oldParent must no longer reference the text")
	})

	t.Run("AddChild of a non-text node is rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		pi := doc.CreatePI("target", "data")

		err := pi.AddChild(mustCreateElement(t, doc, "child"))
		require.Error(t, err, "adding an element child to a PI must be rejected")
		require.Nil(t, pi.FirstChild(), "PI must not gain a child")
		require.Equal(t, "data", string(pi.Content()), "PI data must be unchanged")
	})

	t.Run("AddChild rejects a nil operand", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		pi := doc.CreatePI("target", "data")

		require.ErrorIs(t, pi.AddChild(nil), helium.ErrNilNode, "nil operand must be rejected")
	})

	t.Run("PI serializes as <?target data?>", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		pi := doc.CreatePI("target", "")
		require.NoError(t, pi.AppendText([]byte("data")), "AppendText must succeed")
		require.NoError(t, root.AddChild(pi), "PI must attach under root")

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, "<?target data?>", "PI must serialize from its data string")
	})
}

// TestAddChildNonMutableOperand verifies that inserting an already-linked
// non-MutableNode operand (NamespaceNodeWrapper) detaches it from its old parent
// rather than silently skipping the unlink and leaving a stale link.
func TestAddChildNonMutableOperand(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	src := mustCreateElement(t, doc, "src")
	dst := mustCreateElement(t, doc, "dst")
	ns := helium.NewNamespace("p", "urn:example")
	nsw := helium.NewNamespaceNodeWrapper(ns, nil)

	// Link the wrapper (a non-MutableNode Node) under src first.
	require.NoError(t, src.AddChild(nsw), "wrapper links under src")
	require.Equal(t, helium.Node(src), nsw.Parent(), "wrapper parent is src")
	require.Equal(t, helium.Node(nsw), src.FirstChild(), "src firstChild is wrapper")

	// Moving the wrapper into dst must auto-unlink it from src. Previously the
	// preflight skipped the unlink for non-MutableNode operands, leaving src
	// still pointing at the wrapper.
	require.NoError(t, dst.AddChild(nsw), "wrapper move into dst succeeds")

	require.Equal(t, helium.Node(dst), nsw.Parent(), "wrapper parent is now dst")
	require.Equal(t, helium.Node(nsw), dst.FirstChild(), "dst firstChild is wrapper")
	require.Nil(t, src.FirstChild(), "src no longer holds the wrapper")
	require.Nil(t, src.LastChild(), "src no longer holds the wrapper")
	require.Nil(t, nsw.PrevSibling(), "wrapper has no stale prev")
	require.Nil(t, nsw.NextSibling(), "wrapper has no stale next")
	requireNoCycle(t, dst)
	requireNoCycle(t, src)
}

// TestAddSiblingNonMutableOperand verifies AddSibling auto-unlinks an
// already-linked non-MutableNode operand (NamespaceNodeWrapper) from its old
// parent instead of silently skipping the unlink.
func TestAddSiblingNonMutableOperand(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	src := mustCreateElement(t, doc, "src")
	dst := mustCreateElement(t, doc, "dst")
	anchor := mustCreateElement(t, doc, "anchor")
	ns := helium.NewNamespace("p", "urn:example")
	nsw := helium.NewNamespaceNodeWrapper(ns, nil)

	require.NoError(t, dst.AddChild(anchor), "anchor starts under dst")
	require.NoError(t, src.AddChild(nsw), "wrapper starts under src")

	require.NoError(t, anchor.AddSibling(nsw), "moving wrapper as anchor's sibling succeeds")

	require.Equal(t, helium.Node(dst), nsw.Parent(), "wrapper parent is now dst")
	require.Equal(t, helium.Node(nsw), anchor.NextSibling(), "wrapper follows anchor")
	require.Equal(t, helium.Node(anchor), nsw.PrevSibling(), "anchor precedes wrapper")
	require.Equal(t, helium.Node(nsw), dst.LastChild(), "dst lastChild is wrapper")
	require.Nil(t, src.FirstChild(), "src no longer holds the wrapper")
	require.Nil(t, src.LastChild(), "src no longer holds the wrapper")
	requireNoCycle(t, dst)
	requireNoCycle(t, src)
}

// TestReplaceWithNonMutableOperand verifies Replace splices in a non-MutableNode
// operand (NamespaceNodeWrapper) without panicking on a MutableNode force-cast
// and without leaving stale links on either the replaced node or the operand's
// old parent.
func TestReplaceWithNonMutableOperand(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")
	require.NoError(t, root.AddChild(a), "a starts under root")
	require.NoError(t, root.AddChild(b), "b starts under root")

	src := mustCreateElement(t, doc, "src")
	ns := helium.NewNamespace("p", "urn:example")
	nsw := helium.NewNamespaceNodeWrapper(ns, nil)
	require.NoError(t, src.AddChild(nsw), "wrapper starts under src")

	// Replacing a with the already-linked non-MutableNode wrapper must not panic;
	// the wrapper must be detached from src and take a's position under root.
	require.NoError(t, a.Replace(nsw), "replacing a with the wrapper succeeds")

	require.Equal(t, helium.Node(nsw), root.FirstChild(), "wrapper took a's position")
	require.Equal(t, helium.Node(root), nsw.Parent(), "wrapper parent is root")
	require.Equal(t, helium.Node(b), nsw.NextSibling(), "wrapper precedes b")
	require.Equal(t, helium.Node(nsw), b.PrevSibling(), "b follows the wrapper")
	require.Nil(t, src.FirstChild(), "src no longer holds the wrapper")
	require.Nil(t, src.LastChild(), "src no longer holds the wrapper")
	require.Nil(t, a.Parent(), "replaced node a is detached")
	require.Nil(t, a.NextSibling(), "replaced node a is detached")
	require.Nil(t, a.PrevSibling(), "replaced node a is detached")
	requireNoCycle(t, root)
	requireNoCycle(t, src)
}

// TestReplaceMultipleWithNonMutableOperand covers the multi-operand splice loop
// in replaceNode when one operand is a non-MutableNode wrapper, ensuring the
// loop links operands via base pointers rather than MutableNode setters.
func TestReplaceMultipleWithNonMutableOperand(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	target := mustCreateElement(t, doc, "target")
	require.NoError(t, root.AddChild(target), "target starts under root")

	first := mustCreateElement(t, doc, "first")
	ns := helium.NewNamespace("p", "urn:example")
	nsw := helium.NewNamespaceNodeWrapper(ns, nil)
	last := mustCreateComment(t, doc, []byte("tail"))

	// target is the only child; replacing it with [first, nsw, last] exercises
	// the multi-operand splice and the afterN==nil lastChild update.
	require.NoError(t, target.Replace(first, nsw, last), "multi-operand replace succeeds")

	require.Equal(t, helium.Node(first), root.FirstChild(), "first took target's position")
	require.Equal(t, helium.Node(last), root.LastChild(), "last is the final child")
	require.Equal(t, helium.Node(nsw), first.NextSibling(), "wrapper follows first")
	require.Equal(t, helium.Node(first), nsw.PrevSibling(), "first precedes wrapper")
	require.Equal(t, helium.Node(last), nsw.NextSibling(), "last follows wrapper")
	require.Equal(t, helium.Node(nsw), last.PrevSibling(), "wrapper precedes last")
	require.Equal(t, helium.Node(root), nsw.Parent(), "wrapper parent is root")
	require.Nil(t, target.Parent(), "replaced target is detached")
	requireNoCycle(t, root)
}

// TestPINodeMethods exercises ProcessingInstruction AddChild/AppendText paths
// including the text-merge and rejection branches.
func TestPINodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	pi := doc.CreatePI("target", "data")
	require.Equal(t, "target", pi.Name())
	require.Equal(t, []byte("data"), pi.Content())
	require.Equal(t, helium.ProcessingInstructionNode, pi.Type())

	// Adding a text node merges into the data string.
	require.NoError(t, pi.AddChild(doc.CreateText([]byte(" more"))))
	require.Equal(t, []byte("data more"), pi.Content())

	// Adding a CDATA node also merges.
	require.NoError(t, pi.AddChild(doc.CreateCDATASection([]byte("X"))))
	require.Contains(t, string(pi.Content()), "X")

	// AppendText appends directly.
	require.NoError(t, pi.AppendText([]byte("Y")))
	require.Contains(t, string(pi.Content()), "Y")

	// Adding an element child is rejected.
	require.Error(t, pi.AddChild(doc.CreateElement("e")))

	// Adding a nil node is rejected with ErrNilNode (not a panic).
	require.Error(t, pi.AddChild(nil))
}

// TestCommentNodeMethods exercises Comment AddChild merge/rejection branches.
func TestCommentNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	c := doc.CreateComment([]byte("hello"))
	require.Equal(t, []byte("hello"), c.Content())
	// A comment carries the "(comment)" sentinel name, matching the parenthesized
	// sentinel convention its sibling leaf types (Text "(text)", CDATA "(CDATA)")
	// use for nodes without a real XML name.
	require.Equal(t, "(comment)", c.Name(), "comment sentinel name")
	require.Equal(t, "(comment)", c.LocalName(), "comment sentinel local name")

	// Merging another (unlinked) comment appends its content.
	other := doc.CreateComment([]byte(" world"))
	require.NoError(t, c.AddChild(other))
	require.Equal(t, []byte("hello world"), c.Content())

	// AppendText appends.
	require.NoError(t, c.AppendText([]byte("!")))
	require.Equal(t, []byte("hello world!"), c.Content())

	// Adding a non-comment node is rejected.
	require.Error(t, c.AddChild(doc.CreateText([]byte("t"))))

	// Adding nil is rejected.
	require.Error(t, c.AddChild(nil))
}

// TestCDATANodeMethods exercises the CDATASection node methods.
func TestCDATANodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	cd := doc.CreateCDATASection([]byte("data"))

	// AppendText grows the content.
	require.NoError(t, cd.AppendText([]byte("+more")))
	require.Equal(t, []byte("data+more"), cd.Content())

	// AddChild is rejected on a CDATA node.
	require.Error(t, cd.AddChild(doc.CreateText([]byte("x"))))

	// SetTreeDoc must not panic.
	cd.SetTreeDoc(doc)

	// AddSibling and Replace must run without panicking.
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.AddChild(cd))
	require.NoError(t, cd.AddSibling(doc.CreateCDATASection([]byte("sib"))))
}

// TestEntityRefNodeMethods exercises the EntityRef node-interface methods.
func TestEntityRefNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	ref, err := doc.CreateCharRef("amp")
	require.NoError(t, err)

	ref.SetTreeDoc(doc)
	require.NoError(t, ref.AppendText([]byte("x")))

	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.AddChild(ref))
	require.NoError(t, ref.AddSibling(doc.CreateText([]byte("after"))))
}

// TestEntityNodeMethods covers the Entity node-interface methods.
func TestEntityNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)
	ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	ent.SetOrig("&e;")
	require.False(t, ent.Checked())
	ent.MarkChecked()
	require.True(t, ent.Checked())

	ent.SetTreeDoc(doc)
	require.NoError(t, ent.AppendText([]byte("more")))
}

// TestNotationNodeMethods covers Notation node-interface methods.
func TestNotationNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)
	nota, err := dtd.AddNotation("n", "pub", "sys")
	require.NoError(t, err)

	nota.SetTreeDoc(doc)
	nota.Free()
	require.NoError(t, nota.AppendText([]byte("x")))
}

// TestNotationNodeInterfaceMethods exercises the remaining Notation node-method
// wrappers (AddSibling, Replace, Free, AddChild).
func TestNotationNodeInterfaceMethods(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)
	nota, err := dtd.AddNotation("n", "", "sys")
	require.NoError(t, err)

	// These delegate to the shared tree primitives; exercise without asserting
	// implementation-defined success on an already-attached node.
	_ = nota.AddSibling(doc.CreateElement("x"))
	_ = nota.Replace()
	_ = nota.AddChild(doc.CreateText([]byte("t")))
	nota.Free()
}

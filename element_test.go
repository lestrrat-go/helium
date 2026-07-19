package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func mustCreateElement(t *testing.T, doc *helium.Document, name string) *helium.Element {
	t.Helper()
	e := doc.CreateElement(name)
	return e
}

func mustCreateText(t *testing.T, doc *helium.Document, text []byte) *helium.Text {
	t.Helper()
	n := doc.CreateText(text)
	return n
}

func mustCreateComment(t *testing.T, doc *helium.Document, text []byte) *helium.Comment {
	t.Helper()
	n := doc.CreateComment(text)
	return n
}

func TestElementTree(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e1 := mustCreateElement(t, doc, "root")
	e2 := mustCreateElement(t, doc, "e2")
	e3 := mustCreateElement(t, doc, "e3")
	e4 := mustCreateElement(t, doc, "e4")
	err := e2.SetAttribute("id", "e2")
	require.NoError(t, err)
	err = e3.SetAttribute("id", "e3")
	require.NoError(t, err)
	err = e4.SetAttribute("id", "e4")
	require.NoError(t, err)

	require.NoError(t, e1.AddChild(e2), "e1.AddChild(e2) succeeds")
	require.NoError(t, e1.AddChild(e3), "e1.AddChild(e3) succeeds")
	require.NoError(t, e1.AddChild(e4), "e1.AddChild(e4) succeeds")

	require.Equal(t, e2, e1.FirstChild(), "e1.FirstChild is e2")
	require.Equal(t, e4, e1.LastChild(), "e1.LastChild is e4")

	require.Equal(t, e3, e2.NextSibling(), "e2.NextSibling is e3")
	require.Equal(t, e4, e3.NextSibling(), "e3.NextSibling is e4")
	require.Equal(t, nil, e4.NextSibling(), "e4.NextSibling is nil")

	require.Equal(t, e3, e4.PrevSibling(), "e4.PrevSibling is e3")
	require.Equal(t, e2, e3.PrevSibling(), "e3.PrevSibling is e2")
	require.Equal(t, nil, e2.PrevSibling(), "e2.PrevSibling is nil")

	require.NoError(t, e2.AppendText([]byte("e2")), "e2.AppendText succeeds")
	require.Equal(t, []byte("e2"), e2.Content(), "e2.Content matches")

	for _, e := range []helium.Node{e2, e3, e4} {
		require.Equal(t, e1, e.Parent(), "%s.Parent is e1", e.Name())
	}

	str, err := helium.WriteString(e1)
	require.NoError(t, err, "e1.XMLString succeeds")
	require.Equal(t, `<root><e2 id="e2">e2</e2><e3 id="e3"/><e4 id="e4"/></root>`, str, "e1.XMLString produces expected result")
}

func TestElementContent(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e := mustCreateElement(t, doc, "root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		require.NoError(t, e.AppendText(chunk), "AppendText succeeds")
	}

	require.IsType(t, &helium.Text{}, e.LastChild(), "LastChild is a Text node")

	require.Equal(t, []byte("Hello World!"), e.Content())

	e = mustCreateElement(t, doc, "root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		require.NoError(t, e.AddChild(mustCreateText(t, doc, chunk)), "AddChild succeeds")
	}

	require.IsType(t, &helium.Text{}, e.LastChild(), "LastChild is a Text node")

	require.Equal(t, []byte("Hello World!"), e.Content())
}

func TestAddChildCycleGuard(t *testing.T) {
	t.Parallel()

	t.Run("self insertion is rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := mustCreateElement(t, doc, "root")

		err := e.AddChild(e)
		require.Error(t, err, "AddChild(self) must be rejected")
		require.ErrorContains(t, err, "cannot add a node as a child of itself or one of its descendants")
		require.Nil(t, e.FirstChild(), "tree must not be corrupted")
		require.Nil(t, e.LastChild(), "tree must not be corrupted")
		require.Nil(t, e.Parent(), "tree must not be corrupted")
	})

	t.Run("ancestor insertion is rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")
		mid := mustCreateElement(t, doc, "mid")
		leaf := mustCreateElement(t, doc, "leaf")

		require.NoError(t, root.AddChild(mid))
		require.NoError(t, mid.AddChild(leaf))

		err := leaf.AddChild(root)
		require.Error(t, err, "inserting an ancestor as a descendant must be rejected")

		require.Nil(t, leaf.FirstChild(), "leaf must not gain a child")
		require.Nil(t, root.Parent(), "root must remain the tree root")
		require.Equal(t, root, mid.Parent(), "existing tree must be intact")
		require.Equal(t, mid, leaf.Parent(), "existing tree must be intact")
	})

	t.Run("legal reparent move succeeds", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		a := mustCreateElement(t, doc, "a")
		b := mustCreateElement(t, doc, "b")
		leaf := mustCreateElement(t, doc, "leaf")

		require.NoError(t, a.AddChild(leaf), "leaf starts under a")
		require.Equal(t, a, leaf.Parent(), "leaf parent is a")
		require.Equal(t, leaf, a.FirstChild(), "a firstChild is leaf")

		// Move leaf from a to b. The auto-unlink branch must detach leaf from a
		// before relinking it under b, leaving both subtrees consistent.
		require.NoError(t, b.AddChild(leaf), "reparenting leaf to b succeeds")

		require.Equal(t, b, leaf.Parent(), "leaf parent is now b")
		require.Equal(t, leaf, b.FirstChild(), "b firstChild is leaf")
		require.Equal(t, leaf, b.LastChild(), "b lastChild is leaf")
		require.Nil(t, a.FirstChild(), "a no longer has leaf as firstChild")
		require.Nil(t, a.LastChild(), "a no longer has leaf as lastChild")
		require.Nil(t, leaf.PrevSibling(), "leaf has no stale prev sibling")
		require.Nil(t, leaf.NextSibling(), "leaf has no stale next sibling")

		requireNoCycle(t, b)
	})
}

func TestAddSiblingCycleGuard(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	mid := mustCreateElement(t, doc, "mid")
	leaf := mustCreateElement(t, doc, "leaf")

	require.NoError(t, root.AddChild(mid))
	require.NoError(t, mid.AddChild(leaf))

	// leaf.AddSibling(root) would install root under mid (leaf's parent) while
	// root is leaf's own ancestor, creating a cycle.
	err := leaf.AddSibling(root)
	require.Error(t, err, "adding an ancestor as a sibling must be rejected")
	require.ErrorContains(t, err, "cannot add a node as a sibling of itself or one of its descendants")

	require.Nil(t, root.Parent(), "root must remain the tree root")
	require.Equal(t, root, mid.Parent(), "existing tree must be intact")
	require.Equal(t, mid, leaf.Parent(), "existing tree must be intact")
	require.Nil(t, leaf.NextSibling(), "leaf must not gain a sibling")
	require.Equal(t, leaf, mid.LastChild(), "mid lastChild unchanged")

	requireNoCycle(t, root)
}

func TestReplaceCycleGuard(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	mid := mustCreateElement(t, doc, "mid")
	leaf := mustCreateElement(t, doc, "leaf")

	require.NoError(t, root.AddChild(mid))
	require.NoError(t, mid.AddChild(leaf))

	// leaf.Replace(root) would splice root into leaf's position under mid while
	// root is mid's own ancestor, creating a cycle.
	err := leaf.Replace(root)
	require.Error(t, err, "replacing a node with one of its ancestors must be rejected")
	require.ErrorContains(t, err, "cannot replace a node with one of its own ancestors")

	require.Nil(t, root.Parent(), "root must remain the tree root")
	require.Equal(t, root, mid.Parent(), "existing tree must be intact")
	require.Equal(t, mid, leaf.Parent(), "existing tree must be intact")
	require.Equal(t, leaf, mid.FirstChild(), "leaf must remain mid's child")
	require.Equal(t, leaf, mid.LastChild(), "leaf must remain mid's child")

	requireNoCycle(t, root)
}

func TestAddSiblingSelfRejected(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	child := mustCreateElement(t, doc, "child")
	require.NoError(t, root.AddChild(child))

	err := child.AddSibling(child)
	require.Error(t, err, "AddSibling(self) must be rejected")
	require.ErrorContains(t, err, "cannot add a node as a sibling of itself or one of its descendants")

	require.Equal(t, child, root.FirstChild(), "tree must not be corrupted")
	require.Equal(t, child, root.LastChild(), "tree must not be corrupted")
	require.Nil(t, child.NextSibling(), "child must not gain a sibling")
	require.Nil(t, child.PrevSibling(), "child must not gain a sibling")

	requireNoCycle(t, root)
}

func TestAddSiblingAutoUnlink(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")
	anchor := mustCreateElement(t, doc, "anchor")
	moving := mustCreateElement(t, doc, "moving")

	require.NoError(t, a.AddChild(moving), "moving starts under a")
	require.NoError(t, b.AddChild(anchor), "anchor starts under b")

	// anchor.AddSibling(moving) is a legal move: the auto-unlink preflight must
	// detach moving from a before splicing it after anchor under b.
	require.NoError(t, anchor.AddSibling(moving), "moving anchor's sibling succeeds")

	require.Equal(t, b, moving.Parent(), "moving parent is now b")
	require.Equal(t, moving, anchor.NextSibling(), "moving follows anchor")
	require.Equal(t, anchor, moving.PrevSibling(), "anchor precedes moving")
	require.Equal(t, moving, b.LastChild(), "b lastChild is moving")
	require.Nil(t, a.FirstChild(), "a no longer holds moving")
	require.Nil(t, a.LastChild(), "a no longer holds moving")

	requireNoCycle(t, b)
}

func TestTextAddSiblingSelfRejected(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	txt := mustCreateText(t, doc, []byte("hello"))
	require.NoError(t, root.AddChild(txt))

	// Self-merge must be rejected by the shared guard, not silently double the
	// text content via the text-merge fast path.
	err := txt.AddSibling(txt)
	require.Error(t, err, "Text.AddSibling(self) must be rejected")
	require.ErrorContains(t, err, "cannot add a node as a sibling of itself or one of its descendants")

	require.Equal(t, []byte("hello"), txt.Content(), "content must not be doubled")
	require.Nil(t, txt.NextSibling(), "text must not gain a sibling")
	require.Nil(t, txt.PrevSibling(), "text must not gain a sibling")

	requireNoCycle(t, root)
}

func TestTextAddSiblingMergeUnlinks(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	dst := mustCreateElement(t, doc, "dst")
	src := mustCreateElement(t, doc, "src")
	target := mustCreateText(t, doc, []byte("foo"))
	incoming := mustCreateText(t, doc, []byte("bar"))

	require.NoError(t, dst.AddChild(target), "target starts under dst")
	require.NoError(t, src.AddChild(incoming), "incoming starts under src")

	// Merging an already-linked text node must auto-unlink it from src before
	// merging its content into target, leaving no dangling link under src.
	require.NoError(t, target.AddSibling(incoming), "text merge succeeds")

	require.Equal(t, []byte("foobar"), target.Content(), "content merged")
	require.Nil(t, src.FirstChild(), "incoming detached from src")
	require.Nil(t, src.LastChild(), "incoming detached from src")
	require.Nil(t, incoming.Parent(), "incoming has no stale parent")
	require.Nil(t, incoming.PrevSibling(), "incoming has no stale prev")
	require.Nil(t, incoming.NextSibling(), "incoming has no stale next")

	requireNoCycle(t, dst)
	requireNoCycle(t, src)
}

func TestTextAddChildSelfRejected(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	txt := mustCreateText(t, doc, []byte("hello"))
	require.NoError(t, root.AddChild(txt))

	// Self-merge must be rejected by the shared guard, not silently double the
	// text content via the text-merge fast path.
	err := txt.AddChild(txt)
	require.Error(t, err, "Text.AddChild(self) must be rejected")
	require.ErrorContains(t, err, "cannot add a node as a child of itself or one of its descendants")

	require.Equal(t, []byte("hello"), txt.Content(), "content must not be doubled")
	require.Nil(t, txt.FirstChild(), "text must not gain a child")
	require.Equal(t, txt, root.FirstChild(), "tree must not be corrupted")

	requireNoCycle(t, root)
}

func TestTextAddChildMergeUnlinks(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	dst := mustCreateElement(t, doc, "dst")
	src := mustCreateElement(t, doc, "src")
	target := mustCreateText(t, doc, []byte("foo"))
	incoming := mustCreateText(t, doc, []byte("bar"))

	require.NoError(t, dst.AddChild(target), "target starts under dst")
	require.NoError(t, src.AddChild(incoming), "incoming starts under src")

	// Merging an already-linked text node via AddChild must auto-unlink it from
	// src before merging its content into target.
	require.NoError(t, target.AddChild(incoming), "text merge succeeds")

	require.Equal(t, []byte("foobar"), target.Content(), "content merged")
	require.Nil(t, src.FirstChild(), "incoming detached from src")
	require.Nil(t, src.LastChild(), "incoming detached from src")
	require.Nil(t, incoming.Parent(), "incoming has no stale parent")
	require.Nil(t, incoming.PrevSibling(), "incoming has no stale prev")
	require.Nil(t, incoming.NextSibling(), "incoming has no stale next")

	requireNoCycle(t, dst)
	requireNoCycle(t, src)
}

func TestCommentAddChildSelfRejected(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	comment := mustCreateComment(t, doc, []byte("note"))
	require.NoError(t, root.AddChild(comment))

	// Self-merge must be rejected by the shared guard, not silently double the
	// comment content via the comment-merge fast path.
	err := comment.AddChild(comment)
	require.Error(t, err, "Comment.AddChild(self) must be rejected")
	require.ErrorContains(t, err, "cannot add a node as a child of itself or one of its descendants")

	require.Equal(t, []byte("note"), comment.Content(), "content must not be doubled")
	require.Nil(t, comment.FirstChild(), "comment must not gain a child")
	require.Equal(t, comment, root.FirstChild(), "tree must not be corrupted")

	requireNoCycle(t, root)
}

func TestCommentAddChildMergeUnlinks(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	dst := mustCreateElement(t, doc, "dst")
	src := mustCreateElement(t, doc, "src")
	target := mustCreateComment(t, doc, []byte("foo"))
	incoming := mustCreateComment(t, doc, []byte("bar"))

	require.NoError(t, dst.AddChild(target), "target starts under dst")
	require.NoError(t, src.AddChild(incoming), "incoming starts under src")

	// Merging an already-linked comment node via AddChild must auto-unlink it
	// from src before merging its content into target.
	require.NoError(t, target.AddChild(incoming), "comment merge succeeds")

	require.Equal(t, []byte("foobar"), target.Content(), "content merged")
	require.Nil(t, src.FirstChild(), "incoming detached from src")
	require.Nil(t, src.LastChild(), "incoming detached from src")
	require.Nil(t, incoming.Parent(), "incoming has no stale parent")
	require.Nil(t, incoming.PrevSibling(), "incoming has no stale prev")
	require.Nil(t, incoming.NextSibling(), "incoming has no stale next")

	requireNoCycle(t, dst)
	requireNoCycle(t, src)
}

func TestCDATAAddChildRejected(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	cdata := doc.CreateCDATASection([]byte("payload"))

	// CDATASection has no content-merge path; AddChild is always rejected and
	// must never corrupt the node, including for a self-insertion.
	err := cdata.AddChild(cdata)
	require.Error(t, err, "CDATASection.AddChild must be rejected")

	require.Equal(t, []byte("payload"), cdata.Content(), "content must be untouched")
	require.Nil(t, cdata.FirstChild(), "cdata must not gain a child")

	requireNoCycle(t, cdata)
}

func TestReplaceWithLinkedNode(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")

	require.NoError(t, root.AddChild(a))
	require.NoError(t, root.AddChild(b))

	// Replacing a with its own already-linked sibling b is legal: b must be
	// detached from its current position before taking a's place.
	require.NoError(t, a.Replace(b), "replacing a with linked sibling b succeeds")

	require.Equal(t, b, root.FirstChild(), "b took a's position")
	require.Equal(t, b, root.LastChild(), "b is the only child")
	require.Equal(t, root, b.Parent(), "b parent is root")
	require.Nil(t, b.NextSibling(), "b has no stale next")
	require.Nil(t, b.PrevSibling(), "b has no stale prev")
	require.Nil(t, a.Parent(), "replaced node a is detached")
	require.Nil(t, a.NextSibling(), "replaced node a is detached")
	require.Nil(t, a.PrevSibling(), "replaced node a is detached")

	requireNoCycle(t, root)
}

func TestReplaceDuplicateOperandRejected(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := mustCreateElement(t, doc, "root")
	a := mustCreateElement(t, doc, "a")
	b := mustCreateElement(t, doc, "b")

	require.NoError(t, root.AddChild(a))
	require.NoError(t, root.AddChild(b))

	// a.Replace(b, b) names the same node twice; splicing it into two positions
	// would corrupt its sibling links. It must be rejected before any mutation.
	err := a.Replace(b, b)
	require.Error(t, err, "duplicate replacement operands must be rejected")
	require.ErrorContains(t, err, "cannot replace a node with duplicate replacement operands")

	require.Equal(t, a, root.FirstChild(), "tree must be untouched")
	require.Equal(t, b, root.LastChild(), "tree must be untouched")
	require.Equal(t, b, a.NextSibling(), "a still precedes b")
	require.Equal(t, a, b.PrevSibling(), "b still follows a (no self-link)")
	require.Nil(t, b.NextSibling(), "b must not self-link")

	requireNoCycle(t, root)
}

// requireNoCycle verifies the subtree rooted at n is acyclic using a bounded
// traversal with a visited-node-identity set. It first walks n's parent chain
// (bounded) to confirm n is not its own ancestor, then performs a bounded
// depth-first walk over the descendants and sibling chains, failing if any node
// is visited twice (a cycle) or if the bound is exceeded. Unlike a serialize-
// based detector, this never hangs on a cyclic tree.
func requireNoCycle(t *testing.T, n helium.Node) {
	t.Helper()

	const limit = 10000

	steps := 0
	for anc := n.Parent(); anc != nil; anc = anc.Parent() {
		require.NotSame(t, n, anc, "node must not be its own ancestor")
		steps++
		require.Less(t, steps, limit, "parent chain must be finite")
	}

	visited := make(map[helium.Node]struct{})
	stack := []helium.Node{n}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		_, dup := visited[cur]
		require.False(t, dup, "subtree must not revisit a node (cycle detected)")
		visited[cur] = struct{}{}

		require.Less(t, len(visited), limit, "subtree must be finite")

		siblings := 0
		for child := cur.FirstChild(); child != nil; child = child.NextSibling() {
			stack = append(stack, child)
			siblings++
			require.Less(t, siblings, limit, "sibling chain must be finite")
		}
	}
}

func TestSetTreeDocNonMutableChild(t *testing.T) {
	t.Parallel()

	// A non-MutableNode node (NamespaceNodeWrapper) inserted via AddChild,
	// AddSibling, or Replace must not make SetTreeDoc panic on a MutableNode
	// force-cast; instead its OwnerDocument must be retargeted at the new doc.
	t.Run("via AddChild", func(t *testing.T) {
		t.Parallel()
		src := helium.NewDefaultDocument()
		dst := helium.NewDefaultDocument()
		root := mustCreateElement(t, src, "root")
		ns := helium.NewNamespace("p", "urn:p")
		wrapper := helium.NewNamespaceNodeWrapper(ns, nil)

		require.NoError(t, root.AddChild(wrapper), "AddChild of a non-mutable node succeeds")
		require.NotPanics(t, func() { root.SetTreeDoc(dst) }, "SetTreeDoc must not panic on a non-mutable child")
		require.Equal(t, dst, wrapper.OwnerDocument(), "wrapper doc must be retargeted")
	})

	t.Run("via AddSibling", func(t *testing.T) {
		t.Parallel()
		src := helium.NewDefaultDocument()
		dst := helium.NewDefaultDocument()
		root := mustCreateElement(t, src, "root")
		first := mustCreateElement(t, src, "first")
		require.NoError(t, root.AddChild(first))

		ns := helium.NewNamespace("p", "urn:p")
		wrapper := helium.NewNamespaceNodeWrapper(ns, nil)

		require.NoError(t, first.AddSibling(wrapper), "AddSibling of a non-mutable node succeeds")
		require.NotPanics(t, func() { root.SetTreeDoc(dst) }, "SetTreeDoc must not panic on a non-mutable sibling")
		require.Equal(t, dst, wrapper.OwnerDocument(), "wrapper doc must be retargeted")
	})

	t.Run("via Replace", func(t *testing.T) {
		t.Parallel()
		src := helium.NewDefaultDocument()
		dst := helium.NewDefaultDocument()
		root := mustCreateElement(t, src, "root")
		victim := mustCreateElement(t, src, "victim")
		require.NoError(t, root.AddChild(victim))

		ns := helium.NewNamespace("p", "urn:p")
		wrapper := helium.NewNamespaceNodeWrapper(ns, nil)

		require.NoError(t, victim.Replace(wrapper), "Replace with a non-mutable node succeeds")
		require.NotPanics(t, func() { root.SetTreeDoc(dst) }, "SetTreeDoc must not panic on a non-mutable replacement")
		require.Equal(t, dst, wrapper.OwnerDocument(), "wrapper doc must be retargeted")
	})
}

func TestAddChildNilOperand(t *testing.T) {
	t.Parallel()

	t.Run("literal nil", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")

		err := root.AddChild(nil)
		require.ErrorIs(t, err, helium.ErrNilNode, "AddChild(nil) must return ErrNilNode")
		require.Nil(t, root.FirstChild(), "tree must be untouched")
		require.Nil(t, root.LastChild(), "tree must be untouched")
	})

	t.Run("typed nil", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")
		var typedNil helium.Node = (*helium.Element)(nil)

		err := root.AddChild(typedNil)
		require.ErrorIs(t, err, helium.ErrNilNode, "AddChild(typed-nil) must return ErrNilNode")
		require.Nil(t, root.FirstChild(), "tree must be untouched")
		require.Nil(t, root.LastChild(), "tree must be untouched")
	})
}

func TestAddSiblingNilOperand(t *testing.T) {
	t.Parallel()

	t.Run("literal nil", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")
		child := mustCreateElement(t, doc, "child")
		require.NoError(t, root.AddChild(child))

		err := child.AddSibling(nil)
		require.ErrorIs(t, err, helium.ErrNilNode, "AddSibling(nil) must return ErrNilNode")
		require.Nil(t, child.NextSibling(), "tree must be untouched")
		require.Equal(t, child, root.LastChild(), "tree must be untouched")
	})

	t.Run("typed nil", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")
		child := mustCreateElement(t, doc, "child")
		require.NoError(t, root.AddChild(child))
		var typedNil helium.Node = (*helium.Element)(nil)

		err := child.AddSibling(typedNil)
		require.ErrorIs(t, err, helium.ErrNilNode, "AddSibling(typed-nil) must return ErrNilNode")
		require.Nil(t, child.NextSibling(), "tree must be untouched")
		require.Equal(t, child, root.LastChild(), "tree must be untouched")
	})
}

func TestReplaceNilOperand(t *testing.T) {
	t.Parallel()

	t.Run("literal nil", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")
		victim := mustCreateElement(t, doc, "victim")
		require.NoError(t, root.AddChild(victim))

		err := victim.Replace(nil)
		require.ErrorIs(t, err, helium.ErrNilNode, "Replace(nil) must return ErrNilNode")
		require.Equal(t, victim, root.FirstChild(), "tree must be untouched")
		require.Equal(t, victim, root.LastChild(), "tree must be untouched")
		require.Equal(t, root, victim.Parent(), "tree must be untouched")
	})

	t.Run("typed nil", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")
		victim := mustCreateElement(t, doc, "victim")
		require.NoError(t, root.AddChild(victim))
		var typedNil helium.Node = (*helium.Element)(nil)

		err := victim.Replace(typedNil)
		require.ErrorIs(t, err, helium.ErrNilNode, "Replace(typed-nil) must return ErrNilNode")
		require.Equal(t, victim, root.FirstChild(), "tree must be untouched")
		require.Equal(t, victim, root.LastChild(), "tree must be untouched")
		require.Equal(t, root, victim.Parent(), "tree must be untouched")
	})

	t.Run("nil among valid operands", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := mustCreateElement(t, doc, "root")
		victim := mustCreateElement(t, doc, "victim")
		repl := mustCreateElement(t, doc, "repl")
		require.NoError(t, root.AddChild(victim))

		err := victim.Replace(repl, nil)
		require.ErrorIs(t, err, helium.ErrNilNode, "Replace with a nil operand must return ErrNilNode")
		require.Equal(t, victim, root.FirstChild(), "tree must be untouched")
		require.Equal(t, victim, root.LastChild(), "tree must be untouched")
		require.Nil(t, repl.Parent(), "replacement must not have been spliced in")
	})
}

// TestElementNamespaceMutators exercises the namespace-declaration mutators on
// an element: DeclareNamespace, AddNamespaceDecl, RemoveNamespaceByPrefix,
// SetActiveNamespace and SetNs.
func TestElementNamespaceMutators(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	e := doc.CreateElement("e")

	require.NoError(t, e.DeclareNamespace("p", "urn:p"))
	require.Len(t, e.Namespaces(), 1)

	shared := helium.NewNamespace("q", "urn:q")
	require.NoError(t, e.AddNamespaceDecl(shared))
	require.Len(t, e.Namespaces(), 2)

	require.True(t, e.RemoveNamespaceByPrefix("p"), "existing prefix removed")
	require.False(t, e.RemoveNamespaceByPrefix("absent"), "missing prefix is a no-op")
	require.Len(t, e.Namespaces(), 1)

	require.NoError(t, e.SetActiveNamespace("a", "urn:a"))
	require.Equal(t, "a", e.Prefix())
	require.Equal(t, "urn:a", e.URI())
	require.Equal(t, "a:e", e.Name())

	e.SetNs(shared)
	require.Equal(t, "q", e.Prefix())
	require.Equal(t, "urn:q", e.URI())
}

// TestElementGetAttribute exercises GetAttribute and GetAttributeNS.
func TestElementGetAttribute(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	elem := doc.CreateElement("e")
	err := elem.SetAttribute("plain", "p")
	require.NoError(t, err)

	ns := helium.NewNamespace("x", "urn:x")
	err = elem.SetAttributeNS("nsed", "n", ns)
	require.NoError(t, err)

	v, ok := elem.GetAttribute("plain")
	require.True(t, ok)
	require.Equal(t, "p", v)

	_, ok = elem.GetAttribute("absent")
	require.False(t, ok)

	v, ok = elem.GetAttributeNS("nsed", "urn:x")
	require.True(t, ok)
	require.Equal(t, "n", v)

	_, ok = elem.GetAttributeNS("nsed", "urn:wrong")
	require.False(t, ok)
}

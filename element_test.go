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

func TestElementTree(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e1 := mustCreateElement(t, doc, "root")
	e2 := mustCreateElement(t, doc, "e2")
	e3 := mustCreateElement(t, doc, "e3")
	e4 := mustCreateElement(t, doc, "e4")
	_, err := e2.SetAttribute("id", "e2")
	require.NoError(t, err)
	_, err = e3.SetAttribute("id", "e3")
	require.NoError(t, err)
	_, err = e4.SetAttribute("id", "e4")
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
	require.EqualError(t, err, "cannot add a node as a sibling of itself or one of its descendants")

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
	require.EqualError(t, err, "cannot replace a node with one of its own ancestors")

	require.Nil(t, root.Parent(), "root must remain the tree root")
	require.Equal(t, root, mid.Parent(), "existing tree must be intact")
	require.Equal(t, mid, leaf.Parent(), "existing tree must be intact")
	require.Equal(t, leaf, mid.FirstChild(), "leaf must remain mid's child")
	require.Equal(t, leaf, mid.LastChild(), "leaf must remain mid's child")

	requireNoCycle(t, root)
}

// requireNoCycle verifies the subtree rooted at n is acyclic. It walks n's
// parent chain (bounded) to confirm n is not its own ancestor, then serializes
// the subtree: a cyclic child chain would make serialization recurse forever,
// so a bounded successful serialize confirms the descendants form a finite tree.
func requireNoCycle(t *testing.T, n helium.Node) {
	t.Helper()

	const limit = 1000
	steps := 0
	for anc := n.Parent(); anc != nil; anc = anc.Parent() {
		require.NotEqual(t, n, anc, "node must not be its own ancestor")
		steps++
		require.Less(t, steps, limit, "parent chain must be finite")
	}

	_, err := helium.WriteString(n)
	require.NoError(t, err, "serializing an acyclic subtree must succeed")
}

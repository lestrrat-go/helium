package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// documentTailFixture builds a document holding a comment and a root element, in
// that order, and returns the document with its FIRST child as the append
// anchor. A document's child list is an ordinary sibling chain, so an append
// through that anchor is exactly the workload tailJumpTarget exists to answer in
// O(1).
func documentTailFixture(t *testing.T) (*Document, *Comment) {
	t.Helper()

	doc := NewDefaultDocument()
	comment := doc.CreateComment([]byte("lead"))
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(comment))
	require.NoError(t, doc.AddChild(root))
	require.Equal(t, Node(root), doc.LastChild())

	return doc, comment
}

// TestTailJumpTargetDocumentParent pins which *Document parents may resolve an
// append point from their own lastChild. A document is not excluded by TYPE: it
// is excluded by the one CONDITION that makes its record untrustworthy, which is
// holding a node that claims it as a parent without being on its child list.
func TestTailJumpTargetDocumentParent(t *testing.T) {
	t.Parallel()

	t.Run("an ordinary document resolves its own tail", func(t *testing.T) {
		t.Parallel()

		doc, comment := documentTailFixture(t)
		require.Equal(t, doc.LastChild(), tailJumpTarget(doc, comment.baseDocNode()))
	})

	t.Run("a copied external subset declines", func(t *testing.T) {
		t.Parallel()

		doc, comment := documentTailFixture(t)

		// CopyExtSubset gives the copy the destination document as its parent and
		// leaves it off the child list, so the document can no longer trust its
		// own tail record.
		src := NewDefaultDocument()
		srcDTD := newDTD()
		srcDTD.name = "root"
		src.extSubset = srcDTD
		CopyExtSubset(src, doc)
		require.Equal(t, Node(doc), doc.ExtSubset().Parent())

		require.Nil(t, tailJumpTarget(doc, comment.baseDocNode()))
	})

	t.Run("a claim on one parent leaves every other parent resolving", func(t *testing.T) {
		t.Parallel()

		doc, comment := documentTailFixture(t)

		src := NewDefaultDocument()
		srcDTD := newDTD()
		srcDTD.name = "root"
		src.extSubset = srcDTD
		CopyExtSubset(src, doc)
		require.Nil(t, tailJumpTarget(doc, comment.baseDocNode()),
			"the parent handed the claimant declines")

		// The claimant was handed to the DOCUMENT. Every element the document
		// owns still has its lastChild at the end of its own chain, so each keeps
		// its O(1) resolution: a claim on one parent must never cost the others.
		elem, err := doc.CreateElement("elem")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(elem))
		lead := doc.CreateComment([]byte("lead"))
		trail := doc.CreateComment([]byte("trail"))
		require.NoError(t, elem.AddChild(lead))
		require.NoError(t, elem.AddChild(trail))

		require.Equal(t, Node(trail), tailJumpTarget(elem, lead.baseDocNode()))
	})
}

// TestAppendReplacesAForeignChildList pins the shape that has a firstChild
// claiming nobody, on nodes no document owns. resolveOwnedTail finds no tail
// this parent owns, so the append must start an owned list without changing the
// foreign node's sibling links.
func TestAppendReplacesAForeignChildList(t *testing.T) {
	t.Parallel()

	var standalone *Document
	parent, err := standalone.CreateElement("parent")
	require.NoError(t, err)
	first, err := standalone.CreateElement("first")
	require.NoError(t, err)
	later, err := standalone.CreateElement("later")
	require.NoError(t, err)

	// The firstChild-without-lastChild shape: first claims no parent at all, so
	// nothing on parent's list claims parent.
	setFirstChild(parent, first)
	require.NoError(t, parent.AddChild(later))

	require.Equal(t, Node(later), parent.FirstChild(), "the appended child starts the owned list")
	require.Equal(t, Node(later), parent.LastChild())
	require.Equal(t, Node(parent), later.Parent())
	require.Nil(t, first.NextSibling(), "the foreign node's sibling links stay unchanged")
}

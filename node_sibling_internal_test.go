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

	t.Run("a raw parent claim on the document declines", func(t *testing.T) {
		t.Parallel()

		doc, comment := documentTailFixture(t)

		// The claimed parent IS the document, and the claimant belongs to a
		// DIFFERENT document, so the only node in the write that names doc is the
		// operand. A document's own doc pointer is nil, so reaching the record at
		// all takes resolving the operand's owner by type (owningDocument).
		other := NewDefaultDocument()
		stray, err := other.CreateElement("stray")
		require.NoError(t, err)
		unsafeSetParent(stray, doc)

		require.True(t, doc.untrustedLinks)
		require.Nil(t, tailJumpTarget(doc, comment.baseDocNode()))
	})
}

// TestAdoptRawLinkWritesFromOrphan pins the record that no document owned when
// it was made. A raw write on a fully detached subtree has nowhere to be
// recorded, so it lands on the package-level orphan flag, and the document that
// later adopts the subtree must inherit it.
func TestAdoptRawLinkWritesFromOrphan(t *testing.T) {
	// Not parallel: it asserts on the package-level orphan flag.
	var orphan *Document
	a, err := orphan.CreateElement("a")
	require.NoError(t, err)
	b, err := orphan.CreateElement("b")
	require.NoError(t, err)
	unsafeSetNextSibling(a, b)
	require.True(t, orphanUntrustedLinks.Load(), "a write no document owned is recorded package-wide")

	doc := NewDefaultDocument()
	require.False(t, doc.untrustedLinks)
	a.SetTreeDoc(doc)
	require.True(t, doc.untrustedLinks, "the adopting document inherits the record")
}

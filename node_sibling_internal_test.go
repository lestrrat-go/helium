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

	t.Run("a document holding an off-chain claim declines", func(t *testing.T) {
		t.Parallel()

		doc, comment := documentTailFixture(t)

		// stringToNodeList leaves an entity referenced from an attribute value
		// with a firstChild and no lastChild; setFirstChild is the call that does
		// it. An append onto such a parent detaches that child while the child
		// goes on claiming the parent. Recording the claim is what stops the
		// document trusting a lastChild that a later append through the detached
		// child can move off a child list.
		claimed, err := doc.CreateElement("claimed")
		require.NoError(t, err)
		setFirstChild(claimed, doc.CreateText([]byte("first")))
		require.NoError(t, claimed.AddChild(doc.CreateComment([]byte("q"))))

		require.True(t, doc.offChainClaims)
		require.Nil(t, tailJumpTarget(doc, comment.baseDocNode()))
	})
}

// TestAdoptOffChainClaimWithoutOwner pins the record that no document owned when
// it was made. An off-chain claim created on a still-detached subtree has
// nowhere to be recorded, so it lands on the package-level flag, and the
// document that later adopts the subtree must inherit it.
func TestAdoptOffChainClaimWithoutOwner(t *testing.T) {
	// Not parallel: it asserts on the package-level unowned-claim flag.
	var standalone *Document
	parent, err := standalone.CreateElement("parent")
	require.NoError(t, err)
	first, err := standalone.CreateElement("first")
	require.NoError(t, err)
	later, err := standalone.CreateElement("later")
	require.NoError(t, err)

	// The firstChild-without-lastChild shape, on nodes no document owns.
	setFirstChild(parent, first)
	require.NoError(t, parent.AddChild(later))
	require.True(t, unownedOffChainClaim.Load(), "a claim no document owned is recorded package-wide")

	doc := NewDefaultDocument()
	require.False(t, doc.offChainClaims)
	parent.SetTreeDoc(doc)
	require.True(t, doc.offChainClaims, "the adopting document inherits the record")
}

// TestOffChainClaimTravelsWithTheNode pins the half of holdsOffChainClaim's
// argument that is about MOVEMENT. The record is written on the document owning
// the node whose parenthood is claimed, and holdsOffChainClaim reads that same
// node's document back, so the two can only name different documents if the
// claimed node changes owner afterwards. Both directions of that move carry the
// record: to another document, and out of every document.
func TestOffChainClaimTravelsWithTheNode(t *testing.T) {
	// Not parallel: the out-of-every-document arm reads the package-level flag.
	src := NewDefaultDocument()
	claimed, err := src.CreateElement("claimed")
	require.NoError(t, err)

	// The firstChild-without-lastChild shape: appending onto it detaches the
	// child that was there while the child goes on claiming this parent.
	setFirstChild(claimed, src.CreateText([]byte("first")))
	require.NoError(t, claimed.AddChild(src.CreateComment([]byte("q"))))
	require.True(t, src.offChainClaims)
	require.True(t, holdsOffChainClaim(claimed))

	dst := NewDefaultDocument()
	require.False(t, dst.offChainClaims)
	claimed.SetOwnerDocument(dst)
	require.True(t, dst.offChainClaims, "the destination document inherits the record")
	require.True(t, holdsOffChainClaim(claimed), "the guard still declines the claimed node")

	unownedOffChainClaim.Store(false)
	claimed.SetOwnerDocument(nil)
	require.True(t, unownedOffChainClaim.Load(), "leaving every document leaves the record package-wide")
	require.True(t, holdsOffChainClaim(claimed), "which is what a document-less node reads")
}

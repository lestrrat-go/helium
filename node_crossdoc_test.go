package helium

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCrossDocumentMoveSurvivesFree reproduces the corruption where a
// slab-backed node moved from document A into document B is overwritten after
// A.Free() recycles its slab chunks into the global pool. With the fix, A.Free
// is a no-op once a node escaped, so the moved node keeps its content.
func TestCrossDocumentMoveSurvivesFree(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	moved := a.CreateElement("moved")
	txt := a.CreateText([]byte("ORIGINAL-CONTENT"))
	require.NoError(t, moved.AddChild(txt))

	b := NewDocument("1.0", "", StandaloneImplicitNo)
	broot := b.CreateElement("broot")
	require.NoError(t, b.AddChild(broot))

	// Move the subtree from A into B. This must mark A as having escaped storage.
	require.NoError(t, broot.AddChild(moved))
	require.True(t, a.slabEscaped, "moving a node into another document must mark the source as escaped")

	// Free A, then aggressively allocate in a fresh document to reuse any chunk
	// A might have returned to the pool. With the fix A returned nothing.
	a.Free()
	c := NewDocument("1.0", "", StandaloneImplicitNo)
	for range 512 {
		e := c.CreateElement("OVERWRITE")
		tx := c.CreateText([]byte("XXXXXXXXXXXXXXXX"))
		require.NoError(t, e.AddChild(tx))
	}

	require.Equal(t, "moved", moved.Name(), "moved element struct was overwritten by a reused slab chunk")
	require.Equal(t, "ORIGINAL-CONTENT", string(txt.Content()), "moved text content bytes were overwritten by reused slab storage")
}

// TestPlainParseDoesNotEscape guards the common path: a single-document parse
// never marks the document as escaped, so Free still recycles its slab chunks.
func TestPlainParseDoesNotEscape(t *testing.T) {
	src := []byte(`<?xml version="1.0"?><root xmlns:x="urn:x"><a x:k="v">hi &amp; bye</a><b/></root>`)
	doc, err := NewParser().Parse(context.Background(), src)
	require.NoError(t, err)
	require.False(t, doc.slabEscaped, "a plain single-document parse must not mark the document as escaped")
}

// TestSameDocumentMoveDoesNotEscape moving a node within one document is not a
// cross-document escape, so the flag stays clear and Free keeps recycling.
func TestSameDocumentMoveDoesNotEscape(t *testing.T) {
	d := NewDocument("1.0", "", StandaloneImplicitNo)
	root := d.CreateElement("root")
	require.NoError(t, d.AddChild(root))
	a := d.CreateElement("a")
	b := d.CreateElement("b")
	require.NoError(t, root.AddChild(a))
	require.NoError(t, root.AddChild(b))

	// Re-parent a under b, all within document d.
	require.NoError(t, b.AddChild(a))
	require.False(t, d.slabEscaped, "an intra-document move must not mark the document as escaped")
}

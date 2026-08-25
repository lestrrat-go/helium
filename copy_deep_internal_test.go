package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCopyNodeCrossDocumentDoesNotMarkSourceEscaped pins that CopyNode across
// documents never marks the source document's slab escaped. CopyNode never
// links a SOURCE node into the destination tree — copyNode always allocates a
// brand-new node in dst (dst.CreateElement/CreateText/...) and appendCopiedChild
// only ever links those fresh dst-owned nodes — so noteCrossDocumentEscape,
// which addChild's preflight would otherwise have run, has nothing to mark
// here: the source's own nodes never move. A source document that still
// thinks none of its slab-backed storage escaped can Free() and have its
// chunks recycled normally.
func TestCopyNodeCrossDocumentDoesNotMarkSourceEscaped(t *testing.T) {
	src := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
	root, err := src.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, src.AddChild(root))
	child, err := src.CreateElement("child")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))
	require.NoError(t, child.AddChild(src.CreateText([]byte("hi"))))

	require.False(t, src.slabEscaped, "sanity: a fresh single-document tree starts unescaped")

	dst := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
	_, err = CopyNode(root, dst)
	require.NoError(t, err)

	require.False(t, src.slabEscaped, "CopyNode must not mark the source document's slab escaped: it never links a source node into dst")
}

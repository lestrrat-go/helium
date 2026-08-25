package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
	UnsafeSetNextSibling(a, b)
	require.True(t, orphanRawLinkWrites.Load(), "a write no document owned is recorded package-wide")

	doc := NewDefaultDocument()
	require.False(t, doc.rawLinkWrites)
	a.SetTreeDoc(doc)
	require.True(t, doc.rawLinkWrites, "the adopting document inherits the record")
}

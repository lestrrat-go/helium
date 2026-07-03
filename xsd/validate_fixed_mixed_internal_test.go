package xsd

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestMixedInitialValueCyclicEntity verifies the mixed-content fixed scan is
// safe against a cyclic entity graph. A DOM built directly through the public
// API can form a cycle in the entity child-pointer graph: Document.
// CreateReference links the shared Entity node as the reference's child WITHOUT
// setting the entity's parent (its parent stays the DTD), so AddChild's
// ancestor-chain cycle guard cannot see the link and an Entity may become both
// the child and the parent of a reference. The scan must terminate — the memo's
// in-progress marker breaks the cyclic back-edge — rather than recurse forever
// or overflow the stack.
//
// The scan is exercised directly through mixedInitialValue rather than a full
// Validator.Validate: full-document validation walks the whole document —
// including the DTD, where the cycle lives — via helium.Walk, whose iterative
// traversal is not cycle-guarded and grows its work stack without bound on such
// a graph. That is a helium-core issue tracked separately; this test covers the
// scan's own guard.
func TestMixedInitialValueCyclicEntity(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.AddChild(doc.CreateText([]byte("def"))))

	// ref1 enters the graph from root; ref2 is looped back under the shared
	// Entity node, forming a child-pointer cycle Entity <-> ref2 that ref1 walks
	// into. Attaching ref1 first keeps it under root when the cycle is built.
	ref1, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(ref1))

	ref2, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.NoError(t, ent.AddChild(ref2))

	require.NotPanics(t, func() {
		initial, hasChar, hasElem := mixedInitialValue(root, "abc")
		// The cyclic back-edge contributes nothing; the direct text node is the
		// only character content, so the scan terminates with just it.
		require.Equal(t, "def", initial)
		require.True(t, hasChar)
		require.False(t, hasElem)
	})
}

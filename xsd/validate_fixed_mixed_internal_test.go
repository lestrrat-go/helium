package xsd

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestMixedInitialValueDeepAcyclicChain verifies a finite, acyclic entity chain
// far deeper than any recursion cap is scanned in FULL — not rejected as
// "invalid". The scan is iterative (an explicit stack), so a deep chain neither
// overflows the goroutine stack nor trips a depth limit that would over-reject
// valid content expanding exactly to the fixed value.
func TestMixedInitialValueDeepAcyclicChain(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	// e0 is the leaf (Content "x"); e{i}'s materialized expansion is a reference
	// to e{i-1}. Build bottom-up so each reference's target already exists. depth
	// is well past the old 512 recursion cap.
	const depth = 800
	_, err = dtd.AddEntity("e0", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)
	for i := 1; i <= depth; i++ {
		ent, err := dtd.AddEntity(fmt.Sprintf("e%d", i), enum.InternalGeneralEntity, "", "", "")
		require.NoError(t, err)
		ref, err := doc.CreateReference(fmt.Sprintf("e%d", i-1))
		require.NoError(t, err)
		require.NoError(t, ent.AddChild(ref))
	}

	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))
	top, err := doc.CreateReference(fmt.Sprintf("e%d", depth))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(top))

	require.NotPanics(t, func() {
		initial, hasChar, _, invalid := mixedInitialValue(root, "x")
		require.False(t, invalid, "a finite acyclic chain %d deep must not be rejected as invalid", depth)
		require.Equal(t, "x", initial, "the chain expands exactly to the leaf content")
		require.True(t, hasChar)
	})
}

// TestMixedInitialValueCyclicEntity verifies the mixed-content fixed scan is
// safe against a cyclic entity graph AND fails closed on it. A DOM built
// directly through the public API can form a cycle in the entity child-pointer
// graph: Document.CreateReference links the shared Entity node as the
// reference's child WITHOUT setting the entity's parent (its parent stays the
// DTD), so an Entity may become both the child and the parent of a reference.
// The scan must terminate (the memo's in-progress marker breaks the cyclic
// back-edge) and report the initial value as INVALID — the cyclic expansion
// cannot be materialized reliably, so the mixed-fixed check fails closed rather
// than silently dropping the un-scanned content.
//
// The scan is exercised directly through mixedInitialValue rather than a full
// Validator.Validate because the cycle here lives in the DTD's entity graph,
// which a full document walk reaches only incidentally; this test targets the
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
		_, _, _, invalid := mixedInitialValue(root, "abc")
		// The cyclic back-edge is detected and the scan fails closed.
		require.True(t, invalid, "a cyclic entity graph must mark the initial value invalid")
	})
}

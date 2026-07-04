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

// TestMixedFixedCyclicEntityRejectedByAddChild verifies the FIRST line of
// defense against a cyclic entity graph: the tree-insertion guard. A cycle in
// the entity child-pointer graph would previously be built directly through the
// public API — Document.CreateReference links the shared Entity node as the
// reference's child WITHOUT setting the entity's parent (its parent stays the
// DTD), so wiring that reference back under the Entity forms a child-pointer
// cycle. AddChild's cycle guard now REJECTS that wiring, so a cyclic entity DOM
// cannot be constructed via the public API at all — the mixed-fixed scan
// therefore never sees one in practice.
//
// The scan still carries its OWN cycle guard (the memo's in-progress marker →
// invalid) as defense in depth for a graph corrupted through lower-level
// primitives, but that path is unreachable from the xsd package, so this test
// asserts the reachable, primary protection: the insertion is rejected.
func TestMixedFixedCyclicEntityRejectedByAddChild(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))

	ref, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(ref))

	// ref2's child is the shared Entity node; wiring ref2 under that Entity would
	// close a child-pointer cycle Entity -> ref2 -> Entity. AddChild rejects it.
	ref2, err := doc.CreateReference("e")
	require.NoError(t, err)
	err = ent.AddChild(ref2)
	require.Error(t, err, "AddChild must reject building a cyclic entity graph")
	require.Nil(t, ent.FirstChild(), "the rejected reference must not be linked under the Entity")
}

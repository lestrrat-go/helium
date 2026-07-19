package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestNamespacesDefensiveCopy verifies that the exported Namespaces() accessor
// returns a defensive copy of the node's internal nsDefs slice. Mutating the
// returned slice (overwriting or appending) must NOT corrupt the node's
// internal namespace state, and a subsequent read must still return the
// original declarations.
func TestNamespacesDefensiveCopy(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)

	elem, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, elem.DeclareNamespace("a", "urn:a"))
	require.NoError(t, elem.DeclareNamespace("b", "urn:b"))

	first := elem.Namespaces()
	require.Len(t, first, 2, "initial namespace count")
	require.Equal(t, "a", first[0].Prefix())
	require.Equal(t, "b", first[1].Prefix())

	// Overwrite an element of the returned slice. This must not change the
	// node's internal state.
	first[0] = nil

	// Append to the returned slice. If the slice aliases the node's internal
	// backing array (and has spare capacity), this could clobber internal
	// state too.
	first = append(first, nil)
	_ = first

	got := elem.Namespaces()
	require.Len(t, got, 2, "namespace count after caller mutation")
	require.NotNil(t, got[0], "first namespace must be untouched")
	require.Equal(t, "a", got[0].Prefix(), "first namespace prefix after caller mutation")
	require.Equal(t, "b", got[1].Prefix(), "second namespace prefix after caller mutation")

	// Two separate Namespaces() calls must not alias each other either.
	a := elem.Namespaces()
	b := elem.Namespaces()
	a[0] = nil
	require.NotNil(t, b[0], "second Namespaces() call must not alias the first")
}

package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestAddChildRejectsEntityChildCycle covers the cycle that the ancestor-only
// guard cannot see: an entity reference's child is the shared Entity node, whose
// parent pointer stays the DTD (mirroring libxml2 / Document.CreateReference).
// Because the Entity's parent is NOT the reference, adding that reference back
// under the Entity forms a child-pointer cycle Entity -> ref -> Entity that the
// ancestor walk (which follows PARENT pointers from the insertion point) never
// detects. AddChild must reject it so downstream tree walkers cannot loop.
func TestAddChildRejectsEntityChildCycle(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	// CreateReference links the shared Entity as the reference's child without
	// setting the Entity's parent to the reference.
	ref, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.Equal(t, ent, ref.FirstChild(), "reference's child is the shared Entity node")

	// ref's child is ent, so adding ref under ent closes a child-pointer cycle.
	err = ent.AddChild(ref)
	require.Error(t, err, "adding a reference under its own Entity child must be rejected")
	require.EqualError(t, err, "cannot add a node as a child of itself or one of its descendants")

	// The tree must be untouched: ent must not have gained ref as a child.
	require.Nil(t, ent.FirstChild(), "Entity must not gain the reference as a child")
	require.Nil(t, ent.LastChild(), "Entity must not gain the reference as a child")
}

// TestAddChildAllowsLegitimateEntityReference guards against over-rejection: a
// reference whose Entity child does NOT reach the insertion parent is a normal,
// legal insertion and must succeed. This is the shape produced when parsing
// <root>&e;</root>.
func TestAddChildAllowsLegitimateEntityReference(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))

	ref, err := doc.CreateReference("e")
	require.NoError(t, err)

	require.NoError(t, root.AddChild(ref), "a reference whose Entity does not reach root is a legal child")
	require.Equal(t, ref, root.FirstChild(), "reference must be attached under root")
}

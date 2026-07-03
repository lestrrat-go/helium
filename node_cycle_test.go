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

// sharedEntityFixture builds a document whose DTD declares two general entities
// (so the Entity nodes are siblings in the DTD declaration list) and returns a
// root element plus an entity reference to the first entity, attached under
// root. An entity reference's child is the shared first Entity node, whose
// sibling pointers belong to the DTD list — the shape that lets a naive sibling
// walk wander out of the reference's own children into unrelated declarations.
func sharedEntityFixture(t *testing.T) (root *helium.Element, ref *helium.EntityRef, ent helium.Node) {
	t.Helper()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	e1, err := dtd.AddEntity("e1", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)
	_, err = dtd.AddEntity("e2", enum.InternalGeneralEntity, "", "", "y")
	require.NoError(t, err)

	root = doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))

	ref, err = doc.CreateReference("e1")
	require.NoError(t, err)
	require.Equal(t, e1, ref.FirstChild(), "reference's child is the shared first Entity node")
	require.NoError(t, root.AddChild(ref))

	// The second entity is the first entity's sibling in the DTD declaration
	// list, so a walk that followed raw sibling pointers past the shared Entity
	// would reach it.
	require.Equal(t, "e2", e1.NextSibling().Name(),
		"the second entity is the first's DTD sibling — the foreign spill target")

	return root, ref, e1
}

// TestWalkStaysWithinSubtreeAcrossSharedEntity verifies Walk applies the
// owned-boundary rule: descending into a reference's shared Entity child and
// then advancing must NOT follow the Entity's sibling pointer into the DTD's
// unrelated declarations.
func TestWalkStaysWithinSubtreeAcrossSharedEntity(t *testing.T) {
	t.Parallel()

	root, _, _ := sharedEntityFixture(t)

	var visited []string
	err := helium.Walk(root, helium.NodeWalkerFunc(func(n helium.Node) error {
		visited = append(visited, n.Name())
		return nil
	}))
	require.NoError(t, err)
	require.Equal(t, []string{"root", "e1", "e1"}, visited,
		"Walk visits root, the reference (named for e1), and the shared Entity — not the foreign e2 sibling")
	require.NotContains(t, visited, "e2", "Walk must not spill into the DTD's other entity declarations")
}

// TestChildrenRespectOwnedBoundary verifies Children and ChildElements do not
// follow a foreign child's sibling pointers out of the reference's own list.
func TestChildrenRespectOwnedBoundary(t *testing.T) {
	t.Parallel()

	_, ref, ent := sharedEntityFixture(t)

	var kids []helium.Node
	for c := range helium.Children(ref) {
		kids = append(kids, c)
	}
	require.Equal(t, []helium.Node{ent}, kids,
		"Children(ref) yields only the shared Entity, stopping at the owned boundary")
}

// TestDescendantsRespectOwnedBoundary verifies Descendants stays within the
// reference's own subtree across the shared Entity child.
func TestDescendantsRespectOwnedBoundary(t *testing.T) {
	t.Parallel()

	_, ref, ent := sharedEntityFixture(t)

	var got []helium.Node
	for d := range helium.Descendants(ref) {
		got = append(got, d)
	}
	require.Equal(t, []helium.Node{ent}, got,
		"Descendants(ref) yields only the shared Entity, not the DTD siblings")
}

// TestContentStaysWithinOwnedBoundary verifies the aggregating Content() of an
// entity reference returns only its shared Entity's content and does not spill
// into the DTD's following declarations.
func TestContentStaysWithinOwnedBoundary(t *testing.T) {
	t.Parallel()

	_, ref, _ := sharedEntityFixture(t)

	require.Equal(t, []byte("x"), ref.Content(),
		"Content(ref) is the shared Entity's text, not concatenated with foreign DTD siblings")
}

// TestContentTerminatesOnCyclicSiblingList verifies the aggregating Content()
// terminates when a child's sibling pointer forms a cycle.
func TestContentTerminatesOnCyclicSiblingList(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	txt := doc.CreateText([]byte("a"))
	require.NoError(t, root.AddChild(txt))

	// Corrupt the sibling list into a self-cycle.
	txt.SetNextSibling(txt)

	require.Equal(t, []byte("a"), root.Content(),
		"Content must terminate on a cyclic sibling list instead of looping forever")
}

// TestWalkVisitsSharedEntityDAGTwice guards the requirement that Walk does not
// switch to a global visited set: two references to the same entity form a DAG
// where the shared Entity node is reached on two different paths, and Walk must
// visit it on each occurrence rather than deduplicating it away.
func TestWalkVisitsSharedEntityDAGTwice(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)

	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))

	ref1, err := doc.CreateReference("e")
	require.NoError(t, err)
	ref2, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.Equal(t, ent, ref1.FirstChild())
	require.Equal(t, ent, ref2.FirstChild())
	require.NoError(t, root.AddChild(ref1))
	require.NoError(t, root.AddChild(ref2))

	var entityVisits int
	err = helium.Walk(root, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() == helium.EntityNode {
			entityVisits++
		}
		return nil
	}))
	require.NoError(t, err)
	require.Equal(t, 2, entityVisits,
		"the shared Entity reached via two references must be visited twice — no global dedup")
}

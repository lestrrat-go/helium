package xpath_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/stretchr/testify/require"
)

// buildDoc creates a minimal document:
//
//	<root xmlns:ns="urn:ns">
//	  <child ns:attr="v"/>
//	</root>
func buildDoc(t *testing.T) (*helium.Document, *helium.Element, *helium.Element) {
	t.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.DeclareNamespace("ns", "urn:ns"))

	child, err := doc.CreateElement("child")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))

	ns := helium.NewNamespace("ns", "urn:ns")
	require.NoError(t, child.SetAttributeNS("attr", "v", ns))

	return doc, root, child
}

func TestDocOrderCache_Position_Stride(t *testing.T) {
	doc, root, child := buildDoc(t)
	cache := &ixpath.DocOrderCache{}
	cache.BuildFrom(doc)

	docPos := cache.Position(doc)
	rootPos := cache.Position(root)
	childPos := cache.Position(child)

	// Positions use stride 2, so they must be even.
	require.Equal(t, 0, docPos%2, "doc position should be even")
	require.Equal(t, 0, rootPos%2, "root position should be even")
	require.Equal(t, 0, childPos%2, "child position should be even")

	// Document order: doc < root < child.
	require.Less(t, docPos, rootPos)
	require.Less(t, rootPos, childPos)
}

func TestDocOrderCache_Position_NamespaceNode(t *testing.T) {
	_, root, _ := buildDoc(t)
	cache := &ixpath.DocOrderCache{}
	cache.BuildFrom(root.OwnerDocument())

	rootPos := cache.Position(root)

	// Namespace node wrapping the declared namespace on root.
	ns := helium.NewNamespace("ns", "urn:ns")
	nsw := helium.NewNamespaceNodeWrapper(ns, root)

	nsPos := cache.Position(nsw)

	// Namespace node position must be strictly greater than parent element.
	require.Greater(t, nsPos, rootPos, "namespace node should sort after parent element")
	// And it should be parent + 1 (in the stride-2 gap).
	require.Equal(t, rootPos+1, nsPos, "namespace node should be parent + 1")
}

func TestDocumentRoot_NamespaceNode(t *testing.T) {
	doc, root, _ := buildDoc(t)

	ns := helium.NewNamespace("ns", "urn:ns")
	nsw := helium.NewNamespaceNodeWrapper(ns, root)

	got := ixpath.DocumentRoot(nsw)
	require.Equal(t, helium.Node(doc), got, "DocumentRoot of namespace node should be owning document")
}

func TestDocumentRoot_Element(t *testing.T) {
	doc, _, child := buildDoc(t)

	got := ixpath.DocumentRoot(child)
	require.Equal(t, helium.Node(doc), got, "DocumentRoot of child should be owning document")
}

func TestDeduplicateNodes_NamespaceFirst(t *testing.T) {
	_, root, child := buildDoc(t)
	cache := &ixpath.DocOrderCache{}

	ns := helium.NewNamespace("ns", "urn:ns")
	nsw := helium.NewNamespaceNodeWrapper(ns, root)

	// Input: namespace node first, then its parent element, then child.
	// This exercises the case where namespace node appears before parent
	// in the input slice — the stride-2 scheme should still sort correctly.
	nodes := []helium.Node{nsw, root, child}
	result, err := ixpath.DeduplicateNodes(nodes, cache, 100)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Expected document order: root, nsw, child.
	require.Equal(t, helium.Node(root), result[0], "root should come first")
	require.Equal(t, helium.Node(nsw), result[1], "namespace node should come after root")
	require.Equal(t, helium.Node(child), result[2], "child should come last")
}

func TestDeduplicateNodes_DuplicateNamespace(t *testing.T) {
	_, root, _ := buildDoc(t)
	cache := &ixpath.DocOrderCache{}

	ns := helium.NewNamespace("ns", "urn:ns")
	nsw1 := helium.NewNamespaceNodeWrapper(ns, root)
	nsw2 := helium.NewNamespaceNodeWrapper(ns, root)

	// Two wrappers with same parent+prefix should be deduplicated.
	nodes := []helium.Node{nsw1, nsw2, root}
	result, err := ixpath.DeduplicateNodes(nodes, cache, 100)
	require.NoError(t, err)
	require.Len(t, result, 2, "duplicate namespace nodes should be removed")
}

func TestMergeNodeSets_NamespaceFirst(t *testing.T) {
	_, root, child := buildDoc(t)
	cache := &ixpath.DocOrderCache{}

	ns := helium.NewNamespace("ns", "urn:ns")
	nsw := helium.NewNamespaceNodeWrapper(ns, root)

	// Set a has only the namespace node; set b has root and child.
	a := []helium.Node{nsw}
	b := []helium.Node{child, root}
	result, err := ixpath.MergeNodeSets(a, b, cache, 100)
	require.NoError(t, err)
	require.Len(t, result, 3)

	require.Equal(t, helium.Node(root), result[0])
	require.Equal(t, helium.Node(nsw), result[1])
	require.Equal(t, helium.Node(child), result[2])
}

func TestDeduplicateNodes_ExceedsLimit(t *testing.T) {
	_, root, child := buildDoc(t)
	cache := &ixpath.DocOrderCache{}

	nodes := []helium.Node{root, child}
	_, err := ixpath.DeduplicateNodes(nodes, cache, 1)
	require.Error(t, err)
	require.ErrorIs(t, err, ixpath.ErrNodeSetLimit)
}

func TestDocOrderCache_BuildFromMultipleDocuments(t *testing.T) {
	doc1, root1, _ := buildDoc(t)
	doc2, root2, child2 := buildDoc(t)

	cache := &ixpath.DocOrderCache{}
	cache.BuildFrom(doc1)
	cache.BuildFrom(doc2)

	require.NotEqual(t, -1, cache.Position(root1))
	require.NotEqual(t, -1, cache.Position(root2))
	require.NotEqual(t, -1, cache.Position(child2))
	require.Less(t, cache.Compare(root2, child2), 0)
}

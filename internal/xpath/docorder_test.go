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
	_, err = child.SetAttributeNS("attr", "v", ns)
	require.NoError(t, err)

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

// buildWideDoc creates a document whose root has childCount element children,
// returning the children in document order.
func buildWideDoc(t *testing.T, childCount int) []helium.Node {
	t.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	children := make([]helium.Node, 0, childCount)
	for range childCount {
		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(child))
		children = append(children, child)
	}
	return children
}

// dupHeavyInput returns a slice of length inputSize built by repeating a small
// set of distinct nodes, so that after deduplication only len(distinct) unique
// nodes remain. This drives the over-allocation case: the input is large but
// the result is small.
func dupHeavyInput(distinct []helium.Node, inputSize int) []helium.Node {
	out := make([]helium.Node, 0, inputSize)
	for i := range inputSize {
		out = append(out, distinct[i%len(distinct)])
	}
	return out
}

// TestDeduplicateNodes_CapBoundedByMaxNodes verifies that a large, duplicate-
// heavy input does not cause the dedup to allocate a full-input-size result
// buffer. The unique result fits within maxNodes, so the returned slice
// capacity must stay bounded by maxNodes (plus the overflow detection slot),
// not balloon to the input length. This is a memory-efficiency guarantee.
func TestDeduplicateNodes_CapBoundedByMaxNodes(t *testing.T) {
	const inputSize = 1000
	const maxNodes = 4
	children := buildWideDoc(t, maxNodes)
	cache := &ixpath.DocOrderCache{}

	nodes := dupHeavyInput(children, inputSize)
	result, err := ixpath.DeduplicateNodes(nodes, cache, maxNodes)
	require.NoError(t, err)
	require.Len(t, result, maxNodes)
	// Output order/content unchanged: document order of the distinct children.
	for i := range children {
		require.Equal(t, children[i], result[i])
	}
	require.LessOrEqual(t, cap(result), maxNodes+1,
		"result capacity should be bounded by maxNodes, not the input size")
}

// TestDeduplicateNodes_ExceedsLimitLargeInput verifies truncation semantics
// (error, not silent truncation) are preserved for a large over-limit input.
func TestDeduplicateNodes_ExceedsLimitLargeInput(t *testing.T) {
	const inputSize = 1000
	const maxNodes = 4
	children := buildWideDoc(t, inputSize)
	cache := &ixpath.DocOrderCache{}

	result, err := ixpath.DeduplicateNodes(children, cache, maxNodes)
	require.ErrorIs(t, err, ixpath.ErrNodeSetLimit)
	require.Nil(t, result)
}

func TestDeduplicateNodesPreserveOrder_CapBoundedByMaxNodes(t *testing.T) {
	const inputSize = 1000
	const maxNodes = 4
	children := buildWideDoc(t, maxNodes)

	nodes := dupHeavyInput(children, inputSize)
	result, err := ixpath.DeduplicateNodesPreserveOrder(nodes, maxNodes)
	require.NoError(t, err)
	require.Len(t, result, maxNodes)
	for i := range children {
		require.Equal(t, children[i], result[i])
	}
	require.LessOrEqual(t, cap(result), maxNodes+1,
		"result capacity should be bounded by maxNodes, not the input size")
}

func TestDeduplicateNodesPreserveOrder_ExceedsLimitLargeInput(t *testing.T) {
	const inputSize = 1000
	const maxNodes = 4
	children := buildWideDoc(t, inputSize)

	result, err := ixpath.DeduplicateNodesPreserveOrder(children, maxNodes)
	require.ErrorIs(t, err, ixpath.ErrNodeSetLimit)
	require.Nil(t, result)
}

// TestMergeNodeSets_CapBoundedByMaxNodes verifies that a large, duplicate-heavy
// input split across the two merge inputs does not cause MergeNodeSets to grow
// the result buffer past the bound. maxNodes is chosen as a non-power-of-two so
// that lazy append-doubling (the pre-fix behavior, with no preallocation) would
// overshoot to cap 8 for 5 distinct nodes, failing the bound; the preallocated,
// bounded buffer stays at maxNodes+1.
func TestMergeNodeSets_CapBoundedByMaxNodes(t *testing.T) {
	const inputSize = 1000
	const maxNodes = 5
	children := buildWideDoc(t, maxNodes)
	cache := &ixpath.DocOrderCache{}

	// Split a duplicate-heavy input across the two merge inputs. After dedup
	// only the maxNodes distinct children remain.
	all := dupHeavyInput(children, inputSize)
	a := all[:inputSize/2]
	b := all[inputSize/2:]

	result, err := ixpath.MergeNodeSets(a, b, cache, maxNodes)
	require.NoError(t, err)
	require.Len(t, result, maxNodes)
	// Output order/content unchanged: document order of the distinct children.
	for i := range children {
		require.Equal(t, children[i], result[i])
	}
	require.LessOrEqual(t, cap(result), maxNodes+1,
		"result capacity should be bounded by maxNodes, not the input size")
}

// TestMergeNodeSets_ExceedsLimitEarlyExit verifies MergeNodeSets returns
// ErrNodeSetLimit (no silent truncation) and that it does so via early-exit:
// the result buffer never grows past the bounded capacity even when the inputs
// are far larger than the limit.
func TestMergeNodeSets_ExceedsLimitEarlyExit(t *testing.T) {
	const inputSize = 1000
	const maxNodes = 4
	children := buildWideDoc(t, inputSize)
	cache := &ixpath.DocOrderCache{}

	a := children[:inputSize/2]
	b := children[inputSize/2:]

	result, err := ixpath.MergeNodeSets(a, b, cache, maxNodes)
	require.ErrorIs(t, err, ixpath.ErrNodeSetLimit)
	require.Nil(t, result)
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

// childByType returns the first direct child of n whose node type and (when
// non-empty) name match.
func childByType(n helium.Node, typ helium.ElementType, name string) helium.Node {
	for c := range helium.Children(n) {
		if c.Type() != typ {
			continue
		}
		if name == "" || c.Name() == name {
			return c
		}
	}
	return nil
}

// TestDocOrderCache_EntityRefNoSiblingSpill guards the owned-boundary child
// enumeration in indexWalk. An EntityRefNode's child is the shared Entity node
// owned by the DTD, whose sibling pointers thread into the DTD's declaration
// list. Enumerating an entity reference's children via a raw LastChild /
// PrevSibling walk escapes into those sibling declarations and assigns them
// spurious document-order positions AFTER the entity reference, interleaved
// with the element content. indexWalk enumerates via helium.Children, which
// stops at the foreign boundary, so a sibling declaration is never dragged into
// the content region — it keeps only whatever position the DTD-child descent
// already gave it (before the root element), and the real element nodes keep
// correct document order.
//
// The reference is to the SECOND-declared entity (bar) so that a PrevSibling
// walk from its shared node would reach the FIRST declaration (foo); foo is the
// spill victim under the buggy enumeration.
func TestDocOrderCache_EntityRefNoSiblingSpill(t *testing.T) {
	const src = "<?xml version=\"1.0\"?>\n" +
		"<!DOCTYPE root [\n" +
		"<!ENTITY foo \"FOO\">\n" +
		"<!ENTITY bar \"BAR\">\n" +
		"]>\n" +
		"<root><a>before</a>&bar;<b>after</b></root>"

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)

	elemA := childByType(root, helium.ElementNode, "a")
	require.NotNil(t, elemA, "element a")
	elemB := childByType(root, helium.ElementNode, "b")
	require.NotNil(t, elemB, "element b")
	entityRef := childByType(root, helium.EntityRefNode, "bar")
	require.NotNil(t, entityRef, "entity reference bar")

	// The un-referenced sibling declaration foo, found in the DTD.
	dtd := childByType(doc, helium.DTDNode, "")
	if dtd == nil {
		dtd = childByType(doc, helium.DocumentTypeNode, "")
	}
	require.NotNil(t, dtd, "DTD node")
	fooDecl := childByType(dtd, helium.EntityNode, "foo")
	require.NotNil(t, fooDecl, "foo entity declaration")

	cache := &ixpath.DocOrderCache{}
	cache.BuildFrom(doc)

	// Real element nodes keep correct document order regardless of the entity
	// reference between them.
	require.Less(t, cache.Compare(elemA, entityRef), 0, "a before entity ref")
	require.Less(t, cache.Compare(entityRef, elemB), 0, "entity ref before b")
	require.Less(t, cache.Compare(elemA, elemB), 0, "a before b")

	// The spill is gone: the un-referenced declaration foo is NOT positioned
	// after the entity reference. It stays in the DTD region, before the root
	// element. Under the buggy sibling walk, foo landed after the reference,
	// interleaved with the element content (Position(foo) > Position(ref)).
	require.Less(t, cache.Position(fooDecl), cache.Position(root),
		"sibling declaration foo must stay in the DTD region, before the root element")
	require.Less(t, cache.Position(fooDecl), cache.Position(entityRef),
		"sibling declaration foo must not be spilled after the entity reference")
}

// TestDocOrderCache_EntityFreeDocUnaffected locks the byte-identical invariant:
// a document without entity references (and one with a DTD but no references)
// gets exactly the document-order positions the pre-fix indexWalk produced.
func TestDocOrderCache_EntityFreeDocUnaffected(t *testing.T) {
	t.Run("no DTD", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a>x</a><b>y</b></root>`))
		require.NoError(t, err)
		root := doc.DocumentElement()
		a := childByType(root, helium.ElementNode, "a")
		b := childByType(root, helium.ElementNode, "b")

		cache := &ixpath.DocOrderCache{}
		cache.BuildFrom(doc)

		require.Equal(t, 0, cache.Position(doc))
		require.Equal(t, 2, cache.Position(root))
		require.Equal(t, 4, cache.Position(a))
		require.Equal(t, 8, cache.Position(b))
	})

	t.Run("DTD with declarations but no reference", func(t *testing.T) {
		const src = "<?xml version=\"1.0\"?>\n" +
			"<!DOCTYPE root [\n<!ENTITY foo \"FOO\">\n<!ENTITY bar \"BAR\">\n]>\n" +
			"<root><a>x</a><b>y</b></root>"
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		root := doc.DocumentElement()
		a := childByType(root, helium.ElementNode, "a")
		b := childByType(root, helium.ElementNode, "b")

		cache := &ixpath.DocOrderCache{}
		cache.BuildFrom(doc)

		// Declarations index in the DTD region (pre-existing DTD-child descent),
		// before the root element, in both the old and new enumeration.
		require.Equal(t, 8, cache.Position(root))
		require.Equal(t, 10, cache.Position(a))
		require.Equal(t, 14, cache.Position(b))
	})
}

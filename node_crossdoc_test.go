package helium

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCrossDocumentMoveSurvivesFree reproduces the corruption where a
// slab-backed node moved from document A into document B is overwritten after
// A.Free() recycles its slab chunks into the global pool. With the fix, A.Free
// is a no-op once a node escaped, so the moved node keeps its content.
func TestCrossDocumentMoveSurvivesFree(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	moved, err := a.CreateElement("moved")
	require.NoError(t, err)
	txt := a.CreateText([]byte("ORIGINAL-CONTENT"))
	require.NoError(t, moved.AddChild(txt))

	b := NewDocument("1.0", "", StandaloneImplicitNo)
	broot, err := b.CreateElement("broot")
	require.NoError(t, err)
	require.NoError(t, b.AddChild(broot))

	// Move the subtree from A into B. This must mark A as having escaped storage.
	require.NoError(t, broot.AddChild(moved))
	require.True(t, a.slabEscaped, "moving a node into another document must mark the source as escaped")

	// Free A, then aggressively allocate in a fresh document to reuse any chunk
	// A might have returned to the pool. With the fix A returned nothing.
	a.Free()
	c := NewDocument("1.0", "", StandaloneImplicitNo)
	for range 512 {
		e, err := c.CreateElement("OVERWRITE")
		require.NoError(t, err)
		tx := c.CreateText([]byte("XXXXXXXXXXXXXXXX"))
		require.NoError(t, e.AddChild(tx))
	}

	require.Equal(t, "moved", moved.Name(), "moved element struct was overwritten by a reused slab chunk")
	require.Equal(t, "ORIGINAL-CONTENT", string(txt.Content()), "moved text content bytes were overwritten by reused slab storage")
}

// TestPlainParseDoesNotEscape guards the common path: a single-document parse
// never marks the document as escaped, so Free still recycles its slab chunks.
func TestPlainParseDoesNotEscape(t *testing.T) {
	src := []byte(`<?xml version="1.0"?><root xmlns:x="urn:x"><a x:k="v">hi &amp; bye</a><b/></root>`)
	doc, err := NewParser().Parse(context.Background(), src)
	require.NoError(t, err)
	require.False(t, doc.slabEscaped, "a plain single-document parse must not mark the document as escaped")
}

// TestSameDocumentMoveDoesNotEscape moving a node within one document is not a
// cross-document escape, so the flag stays clear and Free keeps recycling.
func TestSameDocumentMoveDoesNotEscape(t *testing.T) {
	d := NewDocument("1.0", "", StandaloneImplicitNo)
	root, err := d.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, d.AddChild(root))
	a, err := d.CreateElement("a")
	require.NoError(t, err)
	b, err := d.CreateElement("b")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(a))
	require.NoError(t, root.AddChild(b))

	// Re-parent a under b, all within document d.
	require.NoError(t, b.AddChild(a))
	require.False(t, d.slabEscaped, "an intra-document move must not mark the document as escaped")
}

// recycleNamespaceSlab allocates enough namespaces in a fresh document to draw
// chunks back out of the shared pool and overwrite any chunk a freed document
// returned to it.
func recycleNamespaceSlab(t *testing.T) {
	t.Helper()
	c := NewDocument("1.0", "", StandaloneImplicitNo)
	for range 2 * slabSize {
		_, err := c.CreateNamespace("q", "urn:overwrite")
		require.NoError(t, err)
	}
}

// TestAddNamespaceDeclCrossDocumentAppendSurvivesFree covers case 1 (append):
// AddNamespaceDecl retains a foreign slab-backed Namespace, so the source
// document's Free must not recycle its slab out from under the retained decl.
func TestAddNamespaceDeclCrossDocumentAppendSurvivesFree(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	el, err := a.CreateElement("el")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(el))

	b := NewDocument("1.0", "", StandaloneImplicitNo)
	ns, err := b.CreateNamespace("p", "urn:new")
	require.NoError(t, err)

	require.NoError(t, el.AddNamespaceDecl(ns)) // case 1: no existing entry for "p" -> append (retains ns)
	require.True(t, b.slabEscaped, "retaining a foreign slab-backed namespace must mark the source escaped")

	b.Free()
	recycleNamespaceSlab(t)

	require.Equal(t, "p", ns.Prefix(), "retained namespace prefix was overwritten by a reused slab chunk")
	require.Equal(t, "urn:new", ns.URI(), "retained namespace URI was overwritten by a reused slab chunk")
	got := el.Namespaces()
	require.Len(t, got, 1)
	require.Equal(t, "p", got[0].Prefix())
	require.Equal(t, "urn:new", got[0].URI())
}

// TestAddNamespaceDeclCrossDocumentCollapseSurvivesFree covers case 3 (collapse):
// AddNamespaceDecl replaces an existing same-prefix slot with the caller's
// foreign slab-backed Namespace, which is likewise retained and must survive the
// source document's Free.
func TestAddNamespaceDeclCrossDocumentCollapseSurvivesFree(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	el, err := a.CreateElement("el")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(el))
	require.NoError(t, el.DeclareNamespace("p", "urn:old")) // A-owned slot

	b := NewDocument("1.0", "", StandaloneImplicitNo)
	ns, err := b.CreateNamespace("p", "urn:new")
	require.NoError(t, err)

	require.NoError(t, el.AddNamespaceDecl(ns)) // case 3: existing "p" at a different URI -> collapse (retains ns)
	require.True(t, b.slabEscaped, "collapsing in a foreign slab-backed namespace must mark the source escaped")

	b.Free()
	recycleNamespaceSlab(t)

	require.Equal(t, "p", ns.Prefix(), "retained namespace prefix was overwritten by a reused slab chunk")
	require.Equal(t, "urn:new", ns.URI(), "retained namespace URI was overwritten by a reused slab chunk")
	got := el.Namespaces()
	require.Len(t, got, 1)
	require.Equal(t, "p", got[0].Prefix())
	require.Equal(t, "urn:new", got[0].URI())
}

// TestAddNamespaceDeclSameDocumentDoesNotEscape a namespace owned by the same
// document as the receiver is not a cross-document escape, so the flag stays
// clear and Free keeps recycling.
func TestAddNamespaceDeclSameDocumentDoesNotEscape(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	el, err := a.CreateElement("el")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(el))
	ns, err := a.CreateNamespace("p", "urn:x")
	require.NoError(t, err)

	require.NoError(t, el.AddNamespaceDecl(ns)) // same document -> no escape
	require.False(t, a.slabEscaped, "a same-document namespace decl must not mark escape")
}

// TestAddNamespaceDeclCrossDocumentNoOpDoesNotEscape a case-2 no-op (an existing
// declaration at the same URI keeps its slot; the caller's object is not
// retained) must not mark the source document escaped.
func TestAddNamespaceDeclCrossDocumentNoOpDoesNotEscape(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	el, err := a.CreateElement("el")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(el))
	require.NoError(t, el.DeclareNamespace("p", "urn:same")) // existing A-owned slot

	b := NewDocument("1.0", "", StandaloneImplicitNo)
	ns, err := b.CreateNamespace("p", "urn:same") // same URI -> case 2 no-op
	require.NoError(t, err)

	require.NoError(t, el.AddNamespaceDecl(ns))
	require.False(t, b.slabEscaped, "a same-URI no-op must not mark the source escaped")
}

// TestSetNamespaceCrossDocumentSurvivesFree covers the active-namespace retention path:
// SetNamespace installs a foreign slab-backed Namespace as the node's active namespace,
// so the source document's Free must not recycle its slab out from under it.
func TestSetNamespaceCrossDocumentSurvivesFree(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	el, err := a.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(el))

	b := NewDocument("1.0", "", StandaloneImplicitNo)
	ns, err := b.CreateNamespace("p", "urn:original")
	require.NoError(t, err)

	el.SetNamespace(ns)
	require.True(t, b.slabEscaped, "installing a foreign slab-backed active namespace must mark the source escaped")

	b.Free()
	recycleNamespaceSlab(t)

	require.Equal(t, "p", ns.Prefix(), "retained active namespace prefix was overwritten by a reused slab chunk")
	require.Equal(t, "urn:original", ns.URI(), "retained active namespace URI was overwritten by a reused slab chunk")
	require.Equal(t, "p:root", el.Name())
}

// TestCreateElementNSCrossDocumentSurvivesFree covers the PR's new constructor
// reaching SetNamespace: an element created in one document with a namespace owned by
// another must keep its prefix/URI after the source document is freed and its
// slab is recycled. Mirrors the audit repro, asserting the observable
// serialization stays <p:root> rather than mutating to the reused binding.
func TestCreateElementNSCrossDocumentSurvivesFree(t *testing.T) {
	dest := NewDocument("1.0", "", StandaloneImplicitNo)
	src := NewDocument("1.0", "", StandaloneImplicitNo)
	ns, err := src.CreateNamespace("p", "urn:original")
	require.NoError(t, err)

	el, err := dest.CreateElementNS("root", ns)
	require.NoError(t, err)
	require.True(t, src.slabEscaped, "CreateElementNS retaining a foreign namespace must mark the source escaped")
	require.NoError(t, dest.SetDocumentElement(el))

	src.Free()
	recycleNamespaceSlab(t)

	require.Equal(t, "p:root", el.Name(), "retained namespace mutated after source Free")
	s, err := WriteString(el)
	require.NoError(t, err)
	require.Contains(t, s, "p:root")
	require.NotContains(t, s, "urn:overwrite")
}

// TestSetNamespaceSameDocumentDoesNotEscape a same-document active namespace is not a
// cross-document escape, so the flag stays clear and Free keeps recycling.
func TestSetNamespaceSameDocumentDoesNotEscape(t *testing.T) {
	a := NewDocument("1.0", "", StandaloneImplicitNo)
	el, err := a.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(el))
	ns, err := a.CreateNamespace("p", "urn:x")
	require.NoError(t, err)

	el.SetNamespace(ns)
	require.False(t, a.slabEscaped, "a same-document active namespace must not mark escape")
}

// TestNamespaceDeclSkipsNilNsDefsEntry proves the per-prefix dedup scan in
// DeclareNamespace/AddNamespaceDecl tolerates a nil nsDefs slot. The public
// AddNamespaceDecl now rejects a nil ns with ErrNilNode, so a nil entry can only
// arise from a direct in-package field write; the scan must skip it rather than
// dereference it.
func TestNamespaceDeclSkipsNilNsDefsEntry(t *testing.T) {
	doc := NewDefaultDocument()
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))

	root.nsDefs = append(root.nsDefs, nil)

	require.NotPanics(t, func() {
		require.NoError(t, root.DeclareNamespace("p", "urn:p"))
	})
	require.NotPanics(t, func() {
		require.NoError(t, root.AddNamespaceDecl(NewNamespace("q", "urn:q")))
	})
}

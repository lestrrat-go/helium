package helium_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestTypedNilNodeHandling covers the reachable typed-nil path: a document with
// no root element yields a typed-nil *Element from DocumentElement(), and the
// public node helpers must treat it as nil rather than panicking.
func TestTypedNilNodeHandling(t *testing.T) {
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	de := doc.DocumentElement() // typed-nil *helium.Element for a rootless doc
	require.Nil(t, de, "DocumentElement of an empty document is a nil *Element")

	// n holds a typed-nil *Element: the interface value is non-nil (it carries a
	// type) even though the pointer is nil, which is the case that used to panic.
	var n helium.Node = de

	t.Run("AsNode reports not-ok for a typed nil", func(t *testing.T) {
		got, ok := helium.AsNode[*helium.Element](n)
		require.False(t, ok, "AsNode must not report ok for a typed-nil pointer")
		require.Nil(t, got)
	})

	t.Run("CopyNode returns ErrNilNode", func(t *testing.T) {
		_, err := helium.CopyNode(n, doc)
		require.ErrorIs(t, err, helium.ErrNilNode)
	})

	t.Run("Children yields nothing", func(t *testing.T) {
		count := 0
		for range helium.Children(n) {
			count++
		}
		require.Zero(t, count)
	})

	t.Run("Walk returns ErrNilNode", func(t *testing.T) {
		err := helium.Walk(n, helium.NodeWalkerFunc(func(helium.Node) error { return nil }))
		require.ErrorIs(t, err, helium.ErrNilNode)
	})

	t.Run("UnlinkNode is a no-op", func(t *testing.T) {
		require.NotPanics(t, func() { helium.UnlinkNode(de) })
	})

	t.Run("ParseInNodeContext returns ErrNilNode", func(t *testing.T) {
		_, err := helium.NewParser().ParseInNodeContext(context.Background(), n, []byte("<a/>"))
		require.ErrorIs(t, err, helium.ErrNilNode)
	})
}

// TestDeclareNamespaceIdempotent covers the duplicate-prefix contract: a prefix
// declared twice with the same URI collapses to one declaration, and a prefix
// redeclared with a different URI updates the single declaration in place.
func TestDeclareNamespaceIdempotent(t *testing.T) {
	t.Run("same URI is a no-op", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el := doc.CreateElement("root")
		require.NoError(t, el.DeclareNamespace("p", "urn:a"))
		require.NoError(t, el.DeclareNamespace("p", "urn:a"))
		nss := el.Namespaces()
		require.Len(t, nss, 1, "duplicate declaration must not append a second xmlns")
		require.Equal(t, "urn:a", nss[0].URI())
	})

	t.Run("different URI updates in place", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el := doc.CreateElement("root")
		require.NoError(t, el.DeclareNamespace("p", "urn:a"))
		require.NoError(t, el.DeclareNamespace("p", "urn:b"))
		nss := el.Namespaces()
		require.Len(t, nss, 1, "a conflicting redeclaration must not append a second xmlns")
		require.Equal(t, "urn:b", nss[0].URI())
	})

	t.Run("distinct prefixes both declared", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el := doc.CreateElement("root")
		require.NoError(t, el.DeclareNamespace("p", "urn:a"))
		require.NoError(t, el.DeclareNamespace("q", "urn:b"))
		require.Len(t, el.Namespaces(), 2)
	})
}

// countXmlnsP returns how many times `xmlns:p="` appears in s.
func countXmlnsP(s string) int {
	return strings.Count(s, `xmlns:p="`)
}

// TestDeclareNamespaceSingleXmlns covers the serialization post-condition: after
// DeclareNamespace returns nil the element emits at most one xmlns:p and the
// output parses back without a duplicate-attribute error, even when an active
// namespace or a pre-existing duplicate declaration used the same prefix.
func TestDeclareNamespaceSingleXmlns(t *testing.T) {
	// FINDING 1: an active namespace object carrying the old URI must not be
	// serialized as a second declaration after the prefix is rebound.
	t.Run("active namespace with same prefix is reconciled", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el := doc.CreateElement("root")
		require.NoError(t, el.DeclareNamespace("p", "urn:old"))
		require.NoError(t, el.SetActiveNamespace("p", "urn:old"))
		require.NoError(t, el.DeclareNamespace("p", "urn:new"))

		nss := el.Namespaces()
		require.Len(t, nss, 1)
		require.Equal(t, "urn:new", nss[0].URI())
		require.NotNil(t, el.Namespace())
		require.Equal(t, "urn:new", el.Namespace().URI(),
			"active namespace must resolve to the surviving declaration")

		out, err := helium.WriteString(el)
		require.NoError(t, err)
		require.Equal(t, 1, countXmlnsP(out),
			"exactly one xmlns:p must be emitted, got: %s", out)

		_, err = helium.NewParser().Parse(context.Background(), []byte(out))
		require.NoError(t, err, "serialized output must parse back cleanly")
	})

	// FINDING 2: two same-prefix declarations added via AddNamespaceDecl must
	// collapse to one after a redeclaration, without mutating the caller objects.
	t.Run("pre-existing duplicate declaration collapses", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el := doc.CreateElement("root")
		first := helium.NewNamespace("p", "urn:first")
		second := helium.NewNamespace("p", "urn:second")
		el.AddNamespaceDecl(first)
		el.AddNamespaceDecl(second)
		require.NoError(t, el.DeclareNamespace("p", "urn:new"))

		nss := el.Namespaces()
		require.Len(t, nss, 1, "later duplicate declaration must be dropped")
		require.Equal(t, "urn:new", nss[0].URI())
		// The caller-owned objects must not have been rewritten in place.
		require.Equal(t, "urn:first", first.URI())
		require.Equal(t, "urn:second", second.URI())

		out, err := helium.WriteString(el)
		require.NoError(t, err)
		require.Equal(t, 1, countXmlnsP(out),
			"exactly one xmlns:p must be emitted, got: %s", out)

		_, err = helium.NewParser().Parse(context.Background(), []byte(out))
		require.NoError(t, err, "serialized output must parse back cleanly")
	})

	// The reject path: an attribute bound to the prefix at a conflicting URI
	// cannot be collapsed, so the redeclaration is rejected and the node is
	// left unchanged.
	t.Run("attribute conflict is rejected", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el := doc.CreateElement("root")
		require.NoError(t, el.DeclareNamespace("p", "urn:old"))
		attrNS := helium.NewNamespace("p", "urn:old")
		_, err := el.SetAttributeNS("attr", "v", attrNS)
		require.NoError(t, err)

		err = el.DeclareNamespace("p", "urn:new")
		require.ErrorIs(t, err, helium.ErrInvalidOperation)

		nss := el.Namespaces()
		require.Len(t, nss, 1)
		require.Equal(t, "urn:old", nss[0].URI(),
			"a rejected redeclaration must leave the declaration unchanged")
	})
}

// TestEmptyReplaceContract covers the empty-Replace contract: an element's
// Replace() with no arguments returns ErrInvalidOperation, matching
// Document.Replace().
func TestEmptyReplaceContract(t *testing.T) {
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	el := doc.CreateElement("root")

	errNode := el.Replace()
	require.ErrorIs(t, errNode, helium.ErrInvalidOperation)

	errDoc := doc.Replace()
	require.ErrorIs(t, errDoc, helium.ErrInvalidOperation)
}

// TestCyclicNodeSentinel covers the matchable sentinels for the guarded
// mutation operations.
func TestCyclicNodeSentinel(t *testing.T) {
	t.Run("AddChild self insertion", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el := doc.CreateElement("root")
		require.ErrorIs(t, el.AddChild(el), helium.ErrCyclicNode)
	})

	t.Run("AddSibling self insertion", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		parent := doc.CreateElement("parent")
		child := doc.CreateElement("child")
		require.NoError(t, parent.AddChild(child))
		require.ErrorIs(t, child.AddSibling(child), helium.ErrCyclicNode)
	})

	t.Run("Replace with an ancestor", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		parent := doc.CreateElement("parent")
		child := doc.CreateElement("child")
		require.NoError(t, parent.AddChild(child))
		require.ErrorIs(t, child.Replace(parent), helium.ErrCyclicNode)
	})
}

package helium_test

import (
	"context"
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

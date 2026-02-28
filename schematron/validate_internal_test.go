package schematron

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
	"github.com/stretchr/testify/require"
)

func TestXpathResultToStringBoolean(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.BooleanResult, Boolean: true}
		require.Equal(t, "True", xpathResultToString(r))
	})
	t.Run("false", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.BooleanResult, Boolean: false}
		require.Equal(t, "False", xpathResultToString(r))
	})
}

func TestXpathResultToStringNodeSet(t *testing.T) {
	doc, err := helium.Parse([]byte(`<root><a/><b/><c/></root>`))
	require.NoError(t, err)

	root := doc.DocumentElement()
	require.NotNil(t, root)

	// Collect child element nodes.
	var children []helium.Node
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			children = append(children, c)
		}
	}
	require.Len(t, children, 3)

	t.Run("empty", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.NodeSetResult, NodeSet: nil}
		require.Equal(t, "", xpathResultToString(r))
	})
	t.Run("single node", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.NodeSetResult, NodeSet: children[:1]}
		require.Equal(t, "a", xpathResultToString(r))
	})
	t.Run("multiple nodes", func(t *testing.T) {
		r := &xpath.Result{Type: xpath.NodeSetResult, NodeSet: children}
		require.Equal(t, "a b c", xpathResultToString(r))
	})
}

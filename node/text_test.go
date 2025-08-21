package node_test

import (
	"testing"

	"github.com/lestrrat-go/helium/node"
	"github.com/stretchr/testify/require"
)

func TestTextAddContent(t *testing.T) {
	n := node.NewText([]byte("Hello "))
	require.NoError(t, n.AddContent([]byte("World!")), "AddContent succeeds")

	buf, err := n.Content(nil)
	require.NoError(t, err, "Content() should succeed")
	require.Equal(t, []byte("Hello World!"), buf, "Content matches")
}

func TestTextAddChild(t *testing.T) {
	n1 := node.NewText([]byte("Hello "))
	n2 := node.NewText([]byte("World!"))

	require.NoError(t, n1.AddChild(n2), "AddChild succeeds")

	buf, err := n1.Content(nil)
	require.NoError(t, err, "Content() should succeed")
	require.Equal(t, []byte("Hello World!"), buf, "Content matches")
}

func TestTextAddChildInvalidNode(t *testing.T) {
	n1 := node.NewText([]byte("Hello "))
	n2 := node.NewProcessingInstruction("target", "data")

	require.Equal(t, node.ErrInvalidOperation, n1.AddChild(n2), "AddChild fails")

	buf, err := n1.Content(nil)
	require.NoError(t, err, "Content() should succeed")
	require.Equal(t, []byte("Hello "), buf, "Content matches")
}

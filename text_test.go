package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextAddContent(t *testing.T) {
	n := newText([]byte("Hello "))
	require.NoError(t, n.AddContent([]byte("World!")), "AddContent succeeds")

	require.Equal(t, []byte("Hello World!"), n.Content(), "Content matches")
}

func TestTextAddChild(t *testing.T) {
	n1 := newText([]byte("Hello "))
	n2 := newText([]byte("World!"))

	require.NoError(t, n1.AddChild(n2), "AddChild succeeds")

	require.Equal(t, []byte("Hello World!"), n1.Content(), "Content matches")
}

func TestTextAddChildInvalidNode(t *testing.T) {
	n1 := newText([]byte("Hello "))
	n2 := &ProcessingInstruction{}

	require.Equal(t, ErrInvalidOperation, n1.AddChild(n2), "AddChild fails")

	require.Equal(t, []byte("Hello "), n1.Content(), "Content matches")
}

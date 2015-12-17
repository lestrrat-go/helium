package helium

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextAddContent(t *testing.T) {
	n := newText([]byte("Hello "))
	if !assert.NoError(t, n.AddContent([]byte("World!")), "AddContent succeeds") {
		return
	}

	if !assert.Equal(t, []byte("Hello World!"), n.Content(), "Content matches") {
		return
	}
}

func TestTextAddChild(t *testing.T) {
	n1 := newText([]byte("Hello "))
	n2 := newText([]byte("World!"))

	if !assert.NoError(t, n1.AddChild(n2), "AddChild succeeds") {
		return
	}

	if !assert.Equal(t, []byte("Hello World!"), n1.Content(), "Content matches") {
		return
	}
}

func TestTextAddChildInvalidNode(t *testing.T) {
	n1 := newText([]byte("Hello "))
	n2 := &Namespace{}

	if !assert.Equal(t, ErrInvalidOperation, n1.AddChild(n2), "AddChild fails") {
		return
	}

	if !assert.Equal(t, []byte("Hello "), n1.Content(), "Content matches") {
		return
	}
}



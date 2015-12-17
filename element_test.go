package helium

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElementContent(t *testing.T) {
	e := newElement("root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		if !assert.NoError(t, e.AddContent(chunk), "AddContent succeeds") {
			return
		}
	}

	if !assert.IsType(t, newText(nil), e.LastChild(), "LastChild is a Text node") {
		return
	}

	if !assert.Equal(t, []byte("Hello World!"), e.Content()) {
		return
	}

	e = newElement("root")
	for _, chunk := range [][]byte{[]byte("Hello "), []byte("World!")} {
		if !assert.NoError(t, e.AddChild(newText(chunk)), "AddChild succeeds") {
			return
		}
	}

	if !assert.IsType(t, newText(nil), e.LastChild(), "LastChild is a Text node") {
		return
	}

	if !assert.Equal(t, []byte("Hello World!"), e.Content()) {
		return
	}

}
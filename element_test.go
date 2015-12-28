package helium

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElementTree(t *testing.T) {
	e1 := newElement("root")
	e2 := newElement("e2")
	e3 := newElement("e3")
	e4 := newElement("e4")
	e2.SetAttribute("id", "e2")
	e3.SetAttribute("id", "e3")
	e4.SetAttribute("id", "e4")

	if !assert.NoError(t, e1.AddChild(e2), "e1.AddChild(e2) succeeds") {
		return
	}
	if !assert.NoError(t, e1.AddChild(e3), "e1.AddChild(e3) succeeds") {
		return
	}
	if !assert.NoError(t, e1.AddChild(e4), "e1.AddChild(e4) succeeds") {
		return
	}

	if !assert.Equal(t, e2, e1.FirstChild(), "e1.FirstChild is e2") {
		return
	}
	if !assert.Equal(t, e4, e1.LastChild(), "e1.LastChild is e4") {
		return
	}

	if !assert.Equal(t, e3, e2.NextSibling(), "e2.NextSibling is e3") {
		return
	}
	if !assert.Equal(t, e4, e3.NextSibling(), "e3.NextSibling is e4") {
		return
	}
	if !assert.Equal(t, nil, e4.NextSibling(), "e4.NextSibling is nil") {
		return
	}

	if !assert.Equal(t, e3, e4.PrevSibling(), "e4.PrevSibling is e3") {
		return
	}
	if !assert.Equal(t, e2, e3.PrevSibling(), "e3.PrevSibling is e2") {
		return
	}
	if !assert.Equal(t, nil, e2.PrevSibling(), "e2.PrevSibling is nil") {
		return
	}

	if !assert.NoError(t, e2.AddContent([]byte("e2")), "e2.AddContent succeeds") {
		return
	}
	if !assert.Equal(t, []byte("e2"), e2.Content(), "e2.Content matches") {
		return
	}

	for _, e := range []Node{e2, e3, e4} {
		if !assert.Equal(t, e1, e.Parent(), "%s.Parent is e1", e.Name()) {
			return
		}
	}

	str, err := e1.XMLString()
	if !assert.NoError(t, err, "e1.XMLString succeeds") {
		return
	}
	if !assert.Equal(t, `<root><e2 id="e2">e2</e2><e3 id="e3"/><e4 id="e4"/></root>`, str, "e1.XMLString produces expected result") {
		return
	}
}

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
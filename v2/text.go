package helium

import "github.com/lestrrat-go/pdebug/v3"

func newText(b []byte) *Text {
	var t Text
	t.content = make([]byte, len(b))
	copy(t.content, b)
	t.name = "(text)"
	return &t
}

func (n *Text) AddChild(v Node) error {
	if t, ok := v.(*Text); ok {
		return n.AddContent(t.content)
	}
	return errInvalidOperation()
}

func (n *Text) AddContent(b []byte) error {
	if pdebug.Enabled {
		g := pdebug.FuncMarker()
		defer g.End()
	}
	n.content = append(n.content, b...)
	return nil
}

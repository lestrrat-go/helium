package helium

import "github.com/lestrrat/go-pdebug"

func newText(b []byte) *Text {
	t := Text{}
	t.etype = TextNode
	t.content = make([]byte, len(b))
	copy(t.content, b)
	t.name = "(text)"
	return &t
}

func (n *Text) AddChild(cur Node) error {
	var t Text
	switch cur.(type) {
	case *Text:
		t = *(cur.(*Text))
	default:
		return ErrInvalidOperation
	}

	return n.AddContent(t.content)
}

func (n *Text) AddContent(b []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Text.AddContent '%s' (%p)", b, n)
		defer func() {
			g.IRelease("END Text.AddContent '%s'", n.content)
		}()
	}
	n.content = append(n.content, b...)
	return nil
}

func (n *Text) AddSibling(cur Node) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Text.AddSibling '%s'", cur.Content())
		defer g.IRelease("END Text.AddSibling")
	}
	if cur.Type() == TextNode {
		return n.AddContent(cur.Content())
	}
	return addSibling(n, cur)
}

func (n Text) Content() []byte {
	return n.content
}

func (n *Text) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *Text) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

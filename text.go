package helium

func newText(b []byte) *Text {
	t := Text{}
	t.typ = TextNode
	t.content = b
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
	n.content = append(n.content, b...)
	return nil
}

func (n Text) Content() []byte {
	return n.content
}
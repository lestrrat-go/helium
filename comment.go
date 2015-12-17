package helium

func newComment(b []byte) *Comment {
	t := Comment{}
	t.typ = CommentNode
	t.content = b
	return &t
}

func (n *Comment) AddChild(cur Node) error {
	var t Comment
	switch cur.(type) {
	case *Comment:
		t = *(cur.(*Comment))
	default:
		return ErrInvalidOperation
	}

	return n.AddContent(t.content)
}

func (n *Comment) AddContent(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

func (n Comment) Content() []byte {
	return n.content
}
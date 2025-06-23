package helium

func newComment(b []byte) *Comment {
	t := Comment{}
	t.etype = CommentNode
	t.content = b
	return &t
}

func (n *Comment) AddChild(cur Node) error {
	if t, ok := cur.(*Comment); ok {
		return n.AddContent(t.content)
	}
	return ErrInvalidOperation
}

func (n *Comment) AddContent(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

// AddSibling adds a new sibling to the end of the sibling nodes.
func (n *Comment) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n Comment) Content() []byte {
	return n.content
}

func (n *Comment) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *Comment) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

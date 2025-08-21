package node

// Comment represents an XML comment node
type Comment struct {
	treeNode
	content []byte
}

func NewComment(content []byte) *Comment {
	return &Comment{
		content: content,
	}
}

func (*Comment) Type() NodeType {
	return CommentNodeType
}

func (*Comment) LocalName() string {
	return "#comment"
}

func (n *Comment) Content(dst []byte) ([]byte, error) {
	return append(dst, n.content...), nil
}

func (n *Comment) AddChild(cur Node) error {
	if c, ok := cur.(*Comment); ok {
		return n.AddContent(c.content)
	}
	return ErrInvalidOperation
}

func (n *Comment) AddContent(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

func (n *Comment) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *Comment) Replace(cur Node) error {
	return replaceNode(n, cur)
}

func (n *Comment) SetNextSibling(sibling Node) error {
	return setNextSibling(n, sibling)
}

func (n *Comment) SetPrevSibling(sibling Node) error {
	return setPrevSibling(n, sibling)
}

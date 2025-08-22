package node

// Text represents a text node in an XML document
type Text struct {
	treeNode
	content []byte
}

func NewText(content []byte) *Text {
	return &Text{
		content: content,
	}
}

func (Text) Type() NodeType {
	return TextNodeType
}

func (n *Text) LocalName() string {
	return "#text"
}

func (n *Text) Content(dst []byte) ([]byte, error) {
	return append(dst, n.content...), nil
}

func (n *Text) AddChild(child Node) error {
	// Text nodes can concatenate with other text nodes
	if child.Type() == TextNodeType {
		childContent, err := child.Content(nil)
		if err != nil {
			return err
		}
		return n.AddContent(childContent)
	}
	return ErrInvalidOperation
}

func (n *Text) AddContent(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

func (n *Text) AddSibling(sibling Node) error {
	return addSibling(n, sibling)
}

func (n *Text) Replace(cur Node) error {
	return replaceNode(n, cur)
}

func (n *Text) SetNextSibling(sibling Node) error {
	return setNextSibling(n, sibling)
}

func (n *Text) SetPrevSibling(sibling Node) error {
	return setPrevSibling(n, sibling)
}

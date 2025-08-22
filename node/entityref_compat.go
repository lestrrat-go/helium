package node

// Compatibility methods for EntityRef to implement Node interface

func (er *EntityRef) Type() NodeType {
	return EntityRefNodeType
}

func (er *EntityRef) LocalName() string {
	return er.name
}

func (er *EntityRef) AddChild(cur Node) error {
	return addChild(er, cur)
}

func (er *EntityRef) AddContent(b []byte) error {
	return addContent(er, b)
}

func (er *EntityRef) AddSibling(cur Node) error {
	return addSibling(er, cur)
}

func (er *EntityRef) Replace(cur Node) error {
	return replaceNode(er, cur)
}

func (er *EntityRef) SetNextSibling(sibling Node) error {
	return setNextSibling(er, sibling)
}

func (er *EntityRef) SetPrevSibling(sibling Node) error {
	return setPrevSibling(er, sibling)
}

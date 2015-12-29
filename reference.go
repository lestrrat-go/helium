package helium

func (e *EntityReference) AddChild(cur Node) error {
	return addChild(e, cur)
}

func (e *EntityReference) AddContent(b []byte) error {
	return addContent(e, b)
}

func (e *EntityReference) AddSibling(cur Node) error {
	return addSibling(e, cur)
}

func (e *EntityReference) Replace(cur Node) {
	replaceNode(e, cur)
}


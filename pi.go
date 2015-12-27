package helium

func (p *ProcessingInstruction) Type() ElementType {
	return ProcessingInstructionNode
}

func (p *ProcessingInstruction) AddChild(cur Node) error {
	return addChild(p, cur)
}

func (p *ProcessingInstruction) AddContent(b []byte) error {
	return addContent(p, b)
}

func (p *ProcessingInstruction) Replace(cur Node) {
	replaceNode(p, cur)
}

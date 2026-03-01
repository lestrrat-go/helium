package helium

type ProcessingInstruction struct {
	docnode
	target string
	data   string
}

func (p *ProcessingInstruction) Name() string {
	return p.target
}

func (p *ProcessingInstruction) Content() []byte {
	return []byte(p.data)
}

func (p *ProcessingInstruction) Type() ElementType {
	return ProcessingInstructionNode
}

func (p *ProcessingInstruction) AddChild(cur Node) error {
	return addChild(p, cur)
}

func (p *ProcessingInstruction) AddContent(b []byte) error {
	return addContent(p, b)
}

func (p *ProcessingInstruction) AddSibling(cur Node) error {
	return addSibling(p, cur)
}

func (p *ProcessingInstruction) Replace(cur Node) error {
	return replaceNode(p, cur)
}

func (p *ProcessingInstruction) SetTreeDoc(doc *Document) {
	setTreeDoc(p, doc)
}

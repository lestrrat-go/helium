package helium

// ProcessingInstruction represents an XML processing instruction node
// (libxml2: xmlNode with type XML_PI_NODE).
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

func (p *ProcessingInstruction) AppendText(b []byte) error {
	return appendText(p, b)
}

func (p *ProcessingInstruction) AddSibling(cur Node) error {
	return addSibling(p, cur)
}

func (p *ProcessingInstruction) Replace(nodes ...Node) error {
	return replaceNode(p, nodes...)
}

func (p *ProcessingInstruction) SetTreeDoc(doc *Document) {
	setTreeDoc(p, doc)
}

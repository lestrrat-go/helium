package node

// ProcessingInstruction represents a processing instruction node
type ProcessingInstructionNode struct {
	treeNode
	target string
	data   string
}

// NewProcessingInstruction creates a new ProcessingInstructionNode
func NewProcessingInstruction(target, data string) *ProcessingInstructionNode {
	return &ProcessingInstructionNode{
		target: target,
		data:   data,
	}
}

func (pi *ProcessingInstructionNode) Type() NodeType {
	return ProcessingInstructionNodeType
}

func (pi *ProcessingInstructionNode) LocalName() string {
	return pi.target
}

func (pi *ProcessingInstructionNode) AddChild(cur Node) error {
	return addChild(pi, cur)
}

func (pi *ProcessingInstructionNode) AddContent(b []byte) error {
	return addContent(pi, b)
}

func (pi *ProcessingInstructionNode) AddSibling(cur Node) error {
	return addSibling(pi, cur)
}

func (pi *ProcessingInstructionNode) Replace(cur Node) error {
	return replaceNode(pi, cur)
}

func (pi *ProcessingInstructionNode) SetNextSibling(sibling Node) error {
	return setNextSibling(pi, sibling)
}

func (pi *ProcessingInstructionNode) SetPrevSibling(sibling Node) error {
	return setPrevSibling(pi, sibling)
}

func (pi *ProcessingInstructionNode) Target() string {
	return pi.target
}

func (pi *ProcessingInstructionNode) Data() string {
	return pi.data
}

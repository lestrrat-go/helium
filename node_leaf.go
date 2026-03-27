package helium

import "github.com/lestrrat-go/pdebug"

// CDATASection represents a CDATA section node in the DOM tree
// (libxml2: xmlNode with type XML_CDATA_SECTION_NODE).
type CDATASection struct {
	node
}

func newCDATASection(b []byte) *CDATASection {
	c := CDATASection{}
	c.etype = CDATASectionNode
	c.content = make([]byte, len(b))
	copy(c.content, b)
	c.name = "(CDATA)"
	return &c
}

func (n *CDATASection) AddChild(cur Node) error {
	return ErrInvalidOperation
}

func (n *CDATASection) AppendText(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

func (n *CDATASection) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n CDATASection) Content() []byte {
	return n.content
}

func (n *CDATASection) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *CDATASection) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

// Comment represents an XML comment node (libxml2: xmlNode with type XML_COMMENT_NODE).
type Comment struct {
	node
}

func newComment(b []byte) *Comment {
	t := Comment{}
	t.etype = CommentNode
	t.content = b
	return &t
}

func (n *Comment) AddChild(cur Node) error {
	if t, ok := cur.(*Comment); ok {
		return n.AppendText(t.content)
	}
	return ErrInvalidOperation
}

func (n *Comment) AppendText(b []byte) error {
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

func (n *Comment) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *Comment) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

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

// EntityRef represents an entity reference node in the DOM tree
// (libxml2: xmlNode with type XML_ENTITY_REF_NODE).
type EntityRef struct {
	node
}

func newEntityRef() *EntityRef {
	n := &EntityRef{}
	n.etype = EntityRefNode
	return n
}

func (e *EntityRef) AddChild(cur Node) error {
	return addChild(e, cur)
}

func (e *EntityRef) AppendText(b []byte) error {
	return appendText(e, b)
}

func (e *EntityRef) AddSibling(cur Node) error {
	return addSibling(e, cur)
}

func (e *EntityRef) Replace(nodes ...Node) error {
	return replaceNode(e, nodes...)
}

func (n *EntityRef) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

// Text represents an XML text node (libxml2: xmlNode with type XML_TEXT_NODE).
type Text struct {
	node
}

const textNodeName = "(text)"

func newText(b []byte) *Text {
	t := Text{}
	t.etype = TextNode
	t.content = make([]byte, len(b))
	copy(t.content, b)
	t.name = textNodeName
	return &t
}

func (n *Text) AddChild(cur Node) error {
	if t, ok := cur.(*Text); ok {
		return n.AppendText(t.content)
	}
	return ErrInvalidOperation
}

func (n *Text) AppendText(b []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Text.AppendText '%s' (%p)", b, n)
		defer func() {
			g.IRelease("END Text.AppendText '%s'", n.content)
		}()
	}
	if doc := n.doc; doc != nil {
		n.content = doc.growOwnedTextContent(n.content, len(b))
	}
	n.content = append(n.content, b...)
	return nil
}

func (n *Text) AddSibling(cur Node) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Text.AddSibling '%s'", cur.Content())
		defer g.IRelease("END Text.AddSibling")
	}
	if cur.Type() == TextNode {
		return n.AppendText(cur.Content())
	}
	return addSibling(n, cur)
}

func (n Text) Content() []byte {
	return n.content
}

func (n *Text) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *Text) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

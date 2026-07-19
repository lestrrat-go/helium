package helium

import (
	"bytes"
	"fmt"
)

// CDATASection represents a CDATA section node in the DOM tree
// (libxml2: xmlNode with type XML_CDATA_SECTION_NODE).
type CDATASection struct {
	node
}

func newCDATASection(b []byte) *CDATASection {
	c := CDATASection{}
	c.etype = CDATASectionNode
	c.content = bytes.Clone(b)
	c.name = "(CDATA)"
	return &c
}

func (n *CDATASection) AddChild(cur Node) error {
	// Reject a nil or typed-nil operand BEFORE the operand-type reference below so
	// the call returns ErrNilNode instead of panicking and leaves the tree
	// untouched.
	if isNilNode(cur) {
		return ErrNilNode
	}
	// A CDATA section carries character data, not child nodes, so no operand kind
	// is a valid child. Reject with the same %w-wrapped ErrInvalidOperation shape
	// the shared addChild uses for an invalid parent, so callers can errors.Is it
	// like every other AddChild rejection.
	return fmt.Errorf("%w: cannot add a %s as a child of a %s node", ErrInvalidOperation, cur.Type(), n.Type())
}

func (n *CDATASection) AppendText(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

func (n *CDATASection) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

// Content returns a defensive copy of the CDATA section's content. Mutating the
// returned slice does NOT affect the node; re-reading returns the original bytes.
func (n CDATASection) Content() []byte {
	return bytes.Clone(n.content)
}

// rawContent returns the internal content slice without copying for read-only
// internal hot paths. See the package-level rawContent for the contract.
func (n CDATASection) rawContent() []byte {
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
	t.content = bytes.Clone(b)
	t.name = "(comment)"
	return &t
}

func (n *Comment) AddChild(cur Node) error {
	// Reject a nil or typed-nil operand BEFORE the type assertion / comment-merge
	// fast path so the call returns ErrNilNode instead of panicking and leaves
	// the tree untouched.
	if isNilNode(cur) {
		return ErrNilNode
	}
	// Run the shared self/cycle guard and auto-unlink BEFORE the comment-merge
	// fast path. Otherwise comment.AddChild(comment) would double its own
	// content, and merging an already-linked comment node would copy its
	// content while leaving it linked under its old parent.
	if t, ok := cur.(*Comment); ok {
		if err := addChildPreflight(n, cur); err != nil {
			return err
		}
		return n.AppendText(t.content)
	}
	// A comment carries character data, not child nodes; only another comment
	// merges (handled above). Reject any other operand with the same %w-wrapped
	// ErrInvalidOperation shape the shared addChild uses for an invalid parent, so
	// callers can errors.Is it like every other AddChild rejection.
	return fmt.Errorf("%w: cannot add a %s as a child of a %s node", ErrInvalidOperation, cur.Type(), n.Type())
}

func (n *Comment) AppendText(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

// AddSibling adds a new sibling to the end of the sibling nodes.
func (n *Comment) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

// Content returns a defensive copy of the comment's content. Mutating the
// returned slice does NOT affect the node; re-reading returns the original bytes.
func (n Comment) Content() []byte {
	return bytes.Clone(n.content)
}

// rawContent returns the internal content slice without copying for read-only
// internal hot paths. See the package-level rawContent for the contract.
func (n Comment) rawContent() []byte {
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

// AddChild on a PI does not attach child nodes. A processing instruction
// carries its content as a string (the "data" portion), not as element/text
// children. This mirrors libxml2, where an xmlNode of type XML_PI_NODE stores
// its content in the node's content string and has no element/text children.
//
// A Text/CDATA child has its content appended to the PI data (mirroring
// xmlNodeAddContent on a PI). Any other node type is rejected, since attaching
// it would corrupt the tree and break serialization.
func (p *ProcessingInstruction) AddChild(cur Node) error {
	// Reject a nil or typed-nil operand BEFORE any method call on cur (a typed
	// nil is a non-nil interface wrapping a nil pointer, so cur.Type() would
	// panic) so the call returns ErrNilNode and leaves the tree untouched.
	if isNilNode(cur) {
		return ErrNilNode
	}
	// An Attribute has no valid placement on a PI (attributes belong on an
	// element). Reject it BEFORE the type switch with the same %w-wrapped
	// ErrInvalidOperation shape the shared addChild uses for a non-element
	// parent, so callers can errors.Is it like every other AddChild rejection.
	if _, ok := cur.(*Attribute); ok {
		return fmt.Errorf("%w: cannot add an attribute as a child of a %s node; attributes belong on an element", ErrInvalidOperation, p.Type())
	}
	switch cur.Type() {
	case TextNode, CDATASectionNode:
		// Run the shared self/cycle guard and auto-unlink BEFORE merging the
		// source content into the PI data. Otherwise merging an already-linked
		// text/CDATA node would copy its content while leaving it linked under
		// its old parent, violating the AddChild auto-unlink contract.
		if err := addChildPreflight(p, cur); err != nil {
			return err
		}
		p.data += string(cur.Content())
		return nil
	default:
		// A self-add (pi.AddChild(pi)) reaches here because a PI is not a text
		// node; detect it by identity so it matches every other leaf self-add
		// (ErrCyclicNode) rather than the generic type rejection below. The
		// check must be identity, NOT wouldCreateCycle: that guard walks the
		// PI's ancestor chain, so it would also match an ancestor operand
		// (pi.AddChild(parentElement)), which — like on the other strict
		// leaves — is just another invalid operand and takes the shared
		// ErrInvalidOperation shape below.
		if cur.baseDocNode() == p.baseDocNode() {
			return fmt.Errorf("%w: cannot add a node as a child of itself or one of its descendants", ErrCyclicNode)
		}
		// Any other node type has no valid placement on a PI. Reject with the same
		// %w-wrapped ErrInvalidOperation shape the shared addChild uses for an
		// invalid parent, so callers can errors.Is it like every other AddChild
		// rejection.
		return fmt.Errorf("%w: cannot add a %s as a child of a %s node", ErrInvalidOperation, cur.Type(), p.Type())
	}
}

// AppendText appends text to the PI's data string rather than creating a child
// text node. See AddChild for rationale.
func (p *ProcessingInstruction) AppendText(b []byte) error {
	p.data += string(b)
	return nil
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
	// fromCharRef records that some of this node's character data originated
	// from a character reference (&#N;/&#xN;) — directly, or via re-parse of a
	// general entity whose replacement text is a character reference. It is used
	// ONLY by element-content validity (VC: Element Valid, XML §3.2.1 as clarified
	// by errata 2e E15): whitespace produced by a character reference does NOT
	// match the S nonterminal, so it is not ignorable in element-only content,
	// unlike literal source whitespace or whitespace from an internal entity whose
	// replacement text is itself literal whitespace. It is invisible to
	// serialization, C14N, XPath string-value, and node copy — none read it — so
	// it changes no output, only the validation verdict.
	fromCharRef bool
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
	// Reject a nil or typed-nil operand BEFORE the type assertion / text-merge
	// fast path so the call returns ErrNilNode instead of panicking and leaves
	// the tree untouched.
	if isNilNode(cur) {
		return ErrNilNode
	}
	// Run the shared self/cycle guard and auto-unlink BEFORE the text-merge
	// fast path. Otherwise txt.AddChild(txt) would double its own content, and
	// merging an already-linked text node would copy its content while leaving
	// it linked under its old parent.
	if t, ok := cur.(*Text); ok {
		if err := addChildPreflight(n, cur); err != nil {
			return err
		}
		n.fromCharRef = n.fromCharRef || t.fromCharRef
		return n.AppendText(t.content)
	}
	// A text node carries character data, not child nodes; only another text node
	// merges (handled above). Reject any other operand with the same %w-wrapped
	// ErrInvalidOperation shape the shared addChild uses for an invalid parent, so
	// callers can errors.Is it like every other AddChild rejection.
	return fmt.Errorf("%w: cannot add a %s as a child of a %s node", ErrInvalidOperation, cur.Type(), n.Type())
}

func (n *Text) AppendText(b []byte) error {
	if doc := n.doc; doc != nil {
		n.content = doc.growOwnedTextContent(n.content, len(b))
	}
	n.content = append(n.content, b...)
	return nil
}

func (n *Text) AddSibling(cur Node) error {
	// Reject a nil or typed-nil operand BEFORE the debug log (which dereferences
	// cur) and the text-merge fast path so the call returns ErrNilNode instead of
	// panicking and leaves the tree untouched.
	if isNilNode(cur) {
		return ErrNilNode
	}
	// Run the shared self/cycle guard and auto-unlink BEFORE the text-merge
	// fast path. Otherwise t.AddSibling(t) would double its own content, and
	// merging an already-linked text node would copy its content while leaving
	// it linked under its old parent.
	if err := addSiblingPreflight(n, cur); err != nil {
		return err
	}
	if cur.Type() == TextNode {
		if t, ok := cur.(*Text); ok {
			n.fromCharRef = n.fromCharRef || t.fromCharRef
		}
		return n.AppendText(cur.Content())
	}
	return addSibling(n, cur)
}

// Content returns a defensive copy of the text node's content. Mutating the
// returned slice does NOT affect the node; re-reading returns the original bytes.
func (n Text) Content() []byte {
	return bytes.Clone(n.content)
}

// rawContent returns the internal content slice without copying for read-only
// internal hot paths. See the package-level rawContent for the contract.
func (n Text) rawContent() []byte {
	return n.content
}

func (n *Text) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *Text) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

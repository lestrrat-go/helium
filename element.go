package helium

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

const _ElementType_name = "ElementNodeAttributeTextNodeCDATASectionNodeEntityRefNodeEntityNodeProcessingInstructionNodeCommentNodeDocumentNodeDocumentTypeNodeDocumentFragNodeNotationNodeHTMLDocumentNodeDTDNodeElementDeclNodeAttributeDeclNodeEntityDeclNodeNamespaceDeclNodeXIncludeStartNodeXIncludeEndNodeNamespaceNode"

var _ElementType_index = [...]uint16{0, 11, 20, 28, 44, 57, 67, 92, 103, 115, 131, 147, 159, 175, 182, 197, 214, 228, 245, 262, 277, 290}

func (i ElementType) String() string {
	i -= 1
	if i < 0 || i >= ElementType(len(_ElementType_index)-1) {
		return fmt.Sprintf("ElementType(%d)", i+1)
	}
	return _ElementType_name[_ElementType_index[i]:_ElementType_index[i+1]]
}
func newElementDecl() *ElementDecl {
	e := ElementDecl{}
	e.etype = ElementDeclNode
	return &e
}

// newElement creates a new Element with the given name
func newElement(name string) *Element {
	e := &Element{}
	e.name = name
	e.etype = ElementNode
	return e
}

func (n *ElementDecl) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *ElementDecl) AddContent(b []byte) error {
	return addContent(n, b)
}

func (n *ElementDecl) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *ElementDecl) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *ElementDecl) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

func (n Element) XMLString() (string, error) {
	out := bytes.Buffer{}
	if err := n.XML(context.Background(), &out); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (n *Element) XML(ctx context.Context, out io.Writer) error {
	return (&Dumper{}).DumpNode(ctx, out, n)
}

// AddChild adds a new child node to the end of the children nodes.
func (n *Element) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *Element) AddContent(b []byte) error {
	return addContent(n, b)
}

// AddSibling adds a new sibling to the end of the sibling nodes.
func (n *Element) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *Element) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *Element) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

func (n *Element) SetAttribute(name, value string) error {
	attr, err := n.doc.CreateAttribute(name, value, nil)
	if err != nil {
		return err
	}

	p := n.properties
	if p == nil {
		n.properties = attr
		return nil
	}

	var last *Attribute
	for ; p != nil; p = p.NextAttribute() {
		if p.Name() == name {
			return ErrDuplicateAttribute
		}

		last = p
	}

	last.SetNextSibling(attr)
	attr.SetPrevSibling(last)

	return nil
}

func (n Element) Attributes() []*Attribute {
	attrs := []*Attribute{}
	for attr := n.properties; attr != nil; {
		attrs = append(attrs, attr)
		if a := attr.NextSibling(); a != nil {
			attr = a.(*Attribute)
		} else {
			attr = nil
		}
	}

	return attrs
}

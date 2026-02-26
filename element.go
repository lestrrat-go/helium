package helium

import (
	"bytes"
	"io"

	"github.com/lestrrat-go/pdebug"
)

// Element is just a wrapper around Node so that we can
// use Go-ish type checks
type Element struct {
	node
}

func newElementDecl() *ElementDecl {
	e := ElementDecl{}
	e.etype = ElementDeclNode
	return &e
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

func newElement(name string) *Element {
	e := Element{}
	e.name = name
	e.etype = ElementNode
	return &e
}

func (n Element) XMLString() (string, error) {
	out := bytes.Buffer{}
	if err := n.XML(&out); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (n *Element) XML(out io.Writer) error {
	return (&Dumper{}).DumpNode(out, n)
}

// AddChild adds a new child node to the end of the children nodes.
func (n *Element) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *Element) AddContent(b []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Element.AddContent '%s'", b)
		defer g.IRelease("END Element.AddContent")
	}
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Element.SetAttribute '%s' (%s)", name, value)
		defer g.IRelease("END Element.SetAttribute")
	}

	attr, err := n.doc.CreateAttribute(name, value, nil)
	if err != nil {
		return err
	}

	n.addProperty(attr)
	return nil
}

// SetLiteralAttribute creates or replaces an attribute with a literal text
// value. Unlike SetAttribute, the value is not parsed for entity references.
// This is useful for HTML where the parser has already resolved entities.
// An empty value creates a text child with empty content (distinguishing
// it from a boolean attribute which has no children).
func (n *Element) SetLiteralAttribute(name, value string) {
	attr := newAttribute(name, nil)
	attr.doc = n.doc
	t := newText([]byte(value))
	t.doc = n.doc
	attr.setFirstChild(t)
	attr.setLastChild(t)
	t.SetParent(attr)
	n.addProperty(attr)
}

// SetBooleanAttribute creates a boolean attribute (name only, no value).
// The attribute has no children, distinguishing it from an attribute with
// an empty string value.
func (n *Element) SetBooleanAttribute(name string) {
	attr := newAttribute(name, nil)
	attr.doc = n.doc
	n.addProperty(attr)
}

// addProperty inserts or replaces an attribute in the element's property list.
func (n *Element) addProperty(attr *Attribute) {
	p := n.properties
	if p == nil {
		n.properties = attr
		attr.SetParent(n)
		return
	}

	var last *Attribute
	for ; p != nil; p = p.NextAttribute() {
		if p.Name() == attr.Name() {
			// Replace existing attribute in-place: splice new attr
			// into the same position in the linked list.
			attr.SetPrevSibling(p.PrevSibling())
			attr.SetNextSibling(p.NextSibling())
			attr.SetParent(n)
			if prev := p.PrevSibling(); prev != nil {
				prev.SetNextSibling(attr)
			}
			if next := p.NextSibling(); next != nil {
				next.SetPrevSibling(attr)
			}
			if n.properties == p {
				n.properties = attr
			}
			// Detach old attribute
			p.SetParent(nil)
			p.SetPrevSibling(nil)
			p.SetNextSibling(nil)
			return
		}

		last = p
	}

	last.SetNextSibling(attr)
	attr.SetPrevSibling(last)
	attr.SetParent(n)
}

// SetAttributeNS creates an attribute with the given local name, value, and namespace.
func (n *Element) SetAttributeNS(localname, value string, ns *Namespace) error {
	attr, err := n.doc.CreateAttribute(localname, value, ns)
	if err != nil {
		return err
	}

	p := n.properties
	if p == nil {
		n.properties = attr
		attr.SetParent(n)
		return nil
	}

	var last *Attribute
	for ; p != nil; p = p.NextAttribute() {
		if p.LocalName() == localname && p.ns == ns {
			return ErrDuplicateAttribute
		}
		last = p
	}

	last.SetNextSibling(attr)
	attr.SetPrevSibling(last)
	attr.SetParent(n)

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
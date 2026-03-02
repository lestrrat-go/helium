package helium

import (
	"bytes"
	"io"

	"github.com/lestrrat-go/pdebug"
)

// Element represents an XML element node (libxml2: xmlNode with type XML_ELEMENT_NODE).
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

func (n *ElementDecl) Replace(cur Node) error {
	return replaceNode(n, cur)
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

// XMLString serializes the element to an XML string using the given options.
func (n Element) XMLString(options ...WriteOption) (string, error) {
	out := bytes.Buffer{}
	if err := n.XML(&out, options...); err != nil {
		return "", err
	}
	return out.String(), nil
}

// XML serializes the element to w using the given options.
func (n *Element) XML(out io.Writer, options ...WriteOption) error {
	return NewWriter(options...).WriteNode(out, n)
}

// AddChild adds a new child node to the end of the children nodes (libxml2: xmlAddChild).
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

// AddSibling adds a new sibling to the end of the sibling nodes (libxml2: xmlAddSibling).
func (n *Element) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *Element) Replace(cur Node) error {
	return replaceNode(n, cur)
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
	setFirstChild(attr, t)
	setLastChild(attr, t)
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

// GetAttribute returns the value of the attribute with the given name,
// or empty string and false if not found.
func (n *Element) GetAttribute(name string) (string, bool) {
	for p := n.properties; p != nil; p = p.NextAttribute() {
		if p.Name() == name {
			return p.Value(), true
		}
	}
	return "", false
}

// HasAttribute reports whether the element has an attribute with the given name.
func (n *Element) HasAttribute(name string) bool {
	_, ok := n.GetAttribute(name)
	return ok
}

// GetAttributeNS returns the value of the attribute with the given
// local name and namespace URI, or empty string and false if not found.
func (n *Element) GetAttributeNS(localName, nsURI string) (string, bool) {
	for p := n.properties; p != nil; p = p.NextAttribute() {
		if p.LocalName() == localName && p.URI() == nsURI {
			return p.Value(), true
		}
	}
	return "", false
}

// GetAttributeNodeNS returns the Attribute node with the given local name and
// namespace URI, or nil if not found. This is the equivalent of libxml2's
// xmlHasNsProp, returning the node itself for further inspection (e.g.,
// checking atype or whether it is a default attribute).
func (n *Element) GetAttributeNodeNS(localName, nsURI string) *Attribute {
	for p := n.properties; p != nil; p = p.NextAttribute() {
		if p.LocalName() == localName && p.URI() == nsURI {
			return p
		}
	}
	return nil
}

// RemoveAttribute removes the attribute with the given name from the element.
// Returns true if an attribute was removed.
func (n *Element) RemoveAttribute(name string) bool {
	for p := n.properties; p != nil; p = p.NextAttribute() {
		if p.Name() == name {
			n.spliceOutAttribute(p)
			return true
		}
	}
	return false
}

// RemoveAttributeNS removes the attribute with the given local name and
// namespace URI. Returns true if an attribute was removed.
func (n *Element) RemoveAttributeNS(localName, nsURI string) bool {
	for p := n.properties; p != nil; p = p.NextAttribute() {
		if p.LocalName() == localName && p.URI() == nsURI {
			n.spliceOutAttribute(p)
			return true
		}
	}
	return false
}

// spliceOutAttribute removes an attribute from the element's property linked list.
func (n *Element) spliceOutAttribute(p *Attribute) {
	if prev := p.PrevSibling(); prev != nil {
		prev.SetNextSibling(p.NextSibling())
	}
	if next := p.NextSibling(); next != nil {
		next.SetPrevSibling(p.PrevSibling())
	}
	if n.properties == p {
		n.properties = p.NextAttribute()
	}
	p.SetParent(nil)
	p.SetPrevSibling(nil)
	p.SetNextSibling(nil)
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
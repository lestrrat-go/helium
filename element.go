package helium

import (
	"bytes"
	"fmt"
	"io"
	"strings"

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

func (n *ElementDecl) AppendText(b []byte) error {
	return appendText(n, b)
}

func (n *ElementDecl) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *ElementDecl) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
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

// XMLString serializes the element to an XML string using the given writer.
func (n Element) XMLString(writers ...Writer) (string, error) {
	out := bytes.Buffer{}
	if err := n.XML(&out, writers...); err != nil {
		return "", err
	}
	return out.String(), nil
}

// XML serializes the element to w using the given writer.
func (n *Element) XML(out io.Writer, writers ...Writer) error {
	writer := NewWriter()
	if len(writers) > 0 {
		writer = writers[0]
	}
	return writer.WriteNode(out, n)
}

// AddChild adds a new child node to the end of the children nodes (libxml2: xmlAddChild).
func (n *Element) AddChild(cur Node) error {
	return addChild(n, cur)
}

// AppendText appends text content to this node (libxml2: xmlNodeAddContent).
func (n *Element) AppendText(b []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Element.AppendText '%s'", b)
		defer g.IRelease("END Element.AppendText")
	}
	return appendText(n, b)
}

// AddSibling adds a new sibling to the end of the sibling nodes (libxml2: xmlAddSibling).
func (n *Element) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *Element) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *Element) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

func (n *Element) SetAttribute(name, value string) (*Element, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Element.SetAttribute '%s' (%s)", name, value)
		defer g.IRelease("END Element.SetAttribute")
	}

	attr, err := n.doc.CreateAttribute(name, value, nil)
	if err != nil {
		return nil, err
	}

	n.addProperty(attr)
	return n, nil
}

// SetLiteralAttribute creates or replaces an attribute with a literal text
// value. Unlike SetAttribute, the value is not parsed for entity references.
// This is useful for HTML where the parser has already resolved entities.
// An empty value creates a text child with empty content (distinguishing
// it from a boolean attribute which has no children).
func (n *Element) SetLiteralAttribute(name, value string) error {
	if strings.ContainsRune(name, ':') {
		return fmt.Errorf("attribute name %q contains a colon: use SetLiteralAttributeNS with a local name and Namespace parameter", name)
	}
	attr := newAttribute(name, nil)
	attr.doc = n.doc
	t := newText([]byte(value))
	t.doc = n.doc
	setFirstChild(attr, t)
	setLastChild(attr, t)
	t.SetParent(attr)
	n.addProperty(attr)
	return nil
}

// SetBooleanAttribute creates a boolean attribute (name only, no value).
// The attribute has no children, distinguishing it from an attribute with
// an empty string value.
func (n *Element) SetBooleanAttribute(name string) error {
	if strings.ContainsRune(name, ':') {
		return fmt.Errorf("attribute name %q contains a colon", name)
	}
	attr := newAttribute(name, nil)
	attr.doc = n.doc
	n.addProperty(attr)
	return nil
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
			pdn := p.baseDocNode()
			attr.prev = pdn.prev
			attr.next = pdn.next
			attr.parent = n
			if prev := pdn.prev; prev != nil {
				prev.baseDocNode().next = attr
			}
			if next := pdn.next; next != nil {
				next.baseDocNode().prev = attr
			}
			if n.properties == p {
				n.properties = attr
			}
			// Detach old attribute
			pdn.parent = nil
			pdn.prev = nil
			pdn.next = nil
			return
		}

		last = p
	}

	last.next = attr
	attr.prev = last
	attr.parent = n
}

// SetLiteralAttributeNS creates or replaces an attribute with a literal text
// value and namespace. Unlike SetAttributeNS, the value is not parsed for
// entity references. This is useful when the parser has already resolved
// entities in attribute values.
func (n *Element) SetLiteralAttributeNS(localname, value string, ns *Namespace) error {
	if strings.ContainsRune(localname, ':') {
		return fmt.Errorf("attribute local name %q contains a colon", localname)
	}
	attr := newAttribute(localname, ns)
	attr.doc = n.doc
	t := newText([]byte(value))
	t.doc = n.doc
	setFirstChild(attr, t)
	setLastChild(attr, t)
	t.SetParent(attr)
	n.addProperty(attr)
	return nil
}

// SetAttributeNS creates an attribute with the given local name, value, and namespace.
func (n *Element) SetAttributeNS(localname, value string, ns *Namespace) (*Element, error) {
	attr, err := n.doc.CreateAttribute(localname, value, ns)
	if err != nil {
		return nil, err
	}

	p := n.properties
	if p == nil {
		n.properties = attr
		attr.SetParent(n)
		return n, nil
	}

	var last *Attribute
	for ; p != nil; p = p.NextAttribute() {
		if p.LocalName() == localname && p.ns == ns {
			return nil, ErrDuplicateAttribute
		}
		last = p
	}

	last.SetNextSibling(attr)
	attr.SetPrevSibling(last)
	attr.SetParent(n)

	return n, nil
}

// AttributePredicate reports whether an attribute matches a lookup.
// Implementations are used by FindAttribute to support alternate
// matching semantics without exposing the property list layout.
type AttributePredicate interface {
	Match(*Attribute) bool
}

// QNamePredicate matches an attribute by QName as returned by Attribute.Name.
type QNamePredicate string

func (p QNamePredicate) Match(a *Attribute) bool {
	return a.Name() == string(p)
}

// LocalNamePredicate matches an attribute by local name only.
// If multiple attributes share the same local name, FindAttribute returns
// the first match in property order.
type LocalNamePredicate string

func (p LocalNamePredicate) Match(a *Attribute) bool {
	return a.LocalName() == string(p)
}

// NSPredicate matches an attribute by local name + namespace URI.
type NSPredicate struct {
	Local        string
	NamespaceURI string
}

func (p NSPredicate) Match(a *Attribute) bool {
	return a.LocalName() == p.Local && a.URI() == p.NamespaceURI
}

// FindAttribute returns the first attribute that matches ap in property order.
// A nil predicate matches nothing and returns nil, false.
func (n *Element) FindAttribute(ap AttributePredicate) (*Attribute, bool) {
	if ap == nil {
		return nil, false
	}
	for p := n.properties; p != nil; p = p.NextAttribute() {
		if ap.Match(p) {
			return p, true
		}
	}
	return nil, false
}

// GetAttribute returns the value of the attribute with the given QName,
// or empty string and false if not found.
func (n *Element) GetAttribute(name string) (string, bool) {
	attr, ok := n.FindAttribute(QNamePredicate(name))
	if !ok {
		return "", false
	}
	return attr.Value(), true
}

// HasAttribute reports whether the element has an attribute with the given name.
func (n *Element) HasAttribute(name string) bool {
	_, ok := n.FindAttribute(QNamePredicate(name))
	return ok
}

// GetAttributeNS returns the value of the attribute with the given
// local name and namespace URI, or empty string and false if not found.
func (n *Element) GetAttributeNS(localName, nsURI string) (string, bool) {
	attr, ok := n.FindAttribute(NSPredicate{Local: localName, NamespaceURI: nsURI})
	if !ok {
		return "", false
	}
	return attr.Value(), true
}

// GetAttributeNodeNS returns the Attribute node with the given local name and
// namespace URI, or nil if not found. This is the equivalent of libxml2's
// xmlHasNsProp, returning the node itself for further inspection (e.g.,
// checking atype or whether it is a default attribute).
func (n *Element) GetAttributeNodeNS(localName, nsURI string) *Attribute {
	attr, ok := n.FindAttribute(NSPredicate{Local: localName, NamespaceURI: nsURI})
	if !ok {
		return nil
	}
	return attr
}

// RemoveAttribute removes the attribute with the given QName from the element.
// Returns true if an attribute was removed.
func (n *Element) RemoveAttribute(name string) bool {
	attr, ok := n.FindAttribute(QNamePredicate(name))
	if !ok {
		return false
	}
	n.spliceOutAttribute(attr)
	return true
}

// RemoveAttributeNS removes the attribute with the given local name and
// namespace URI. Returns true if an attribute was removed.
func (n *Element) RemoveAttributeNS(localName, nsURI string) bool {
	attr, ok := n.FindAttribute(NSPredicate{Local: localName, NamespaceURI: nsURI})
	if !ok {
		return false
	}
	n.spliceOutAttribute(attr)
	return true
}

// spliceOutAttribute removes an attribute from the element's property linked list.
func (n *Element) spliceOutAttribute(p *Attribute) {
	pdn := p.baseDocNode()
	if prev := pdn.prev; prev != nil {
		prev.baseDocNode().next = pdn.next
	}
	if next := pdn.next; next != nil {
		next.baseDocNode().prev = pdn.prev
	}
	if n.properties == p {
		n.properties = p.NextAttribute()
	}
	pdn.parent = nil
	pdn.prev = nil
	pdn.next = nil
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

// ForEachAttribute calls fn for each attribute on the element.
// If fn returns false, iteration stops early.
// This avoids the slice allocation of Attributes().
//
// No dedicated unit tests: iteration order and early-stop semantics
// are exercised transitively by XPath attribute-axis and doc-order tests.
// All current callers always return true; the early-stop path exists as
// a natural consequence of the iterator pattern.
//
// The unchecked type assertion on NextSibling is safe: the properties
// chain is attribute-only by construction (field typed *Attribute,
// only *Attribute nodes are ever linked in).
func (n Element) ForEachAttribute(fn func(*Attribute) bool) {
	for attr := n.properties; attr != nil; {
		if !fn(attr) {
			return
		}
		if a := attr.NextSibling(); a != nil {
			attr = a.(*Attribute)
		} else {
			attr = nil
		}
	}
}

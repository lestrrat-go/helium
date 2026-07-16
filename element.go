package helium

import (
	"fmt"
	"strings"
)

// Element represents an XML element node (libxml2: xmlNode with type XML_ELEMENT_NODE).
type Element struct {
	node
	// contentHasReference records that a reference (a character reference or a
	// general-entity reference) appeared directly in this element's content. A
	// reference is content per XML production [43], so an element declared EMPTY
	// that contains one is invalid (VC: Element Valid, errata 2e E15a) even when
	// the reference expands to nothing and leaves the element childless. It is set
	// by the parser and read ONLY by element-content validity; it is invisible to
	// serialization, C14N, XPath, and copy.
	contentHasReference bool
}

func newElement(name string) *Element {
	e := Element{}
	e.name = name
	e.etype = ElementNode
	return &e
}

// AddChild adds a new child node to the end of the children nodes (libxml2: xmlAddChild).
func (n *Element) AddChild(cur Node) error {
	return addChild(n, cur)
}

// AppendText appends text content to this node (libxml2: xmlNodeAddContent).
func (n *Element) AppendText(b []byte) error {
	return appendText(n, b)
}

// AddSibling adds a new sibling to the end of the sibling nodes (libxml2: xmlAddSibling).
func (n *Element) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

// Replace swaps this element out of its parent, inserting nodes in its place
// (libxml2: xmlReplaceNode). It returns an error if any operand is nil.
func (n *Element) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

// SetTreeDoc sets the owning document of this element and of every node in its
// subtree (libxml2: xmlSetTreeDoc).
func (n *Element) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

// SetAttribute creates or replaces the attribute named name, parsing value for
// entity references, and returns the element for chaining. An existing
// attribute with the same QName is replaced in place. The name must not
// contain a colon; use SetAttributeNS for namespaced attributes, or
// SetLiteralAttribute to store value verbatim without parsing.
func (n *Element) SetAttribute(name, value string) (*Element, error) {
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
	t.parent = attr
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

// attrMatches reports whether existing and the attribute identified by
// (qname, nsURI, localName) are the same attribute for the purposes of
// duplicate detection. Two attributes collide when EITHER their expanded
// name (namespace URI + local name) matches OR their serialized QName
// matches. The expanded-name test catches duplicates declared through
// different namespace prefixes that resolve to the same URI (e.g. p:a and
// q:a both bound to {urn:x}a); the QName test catches duplicates among
// no-namespace attributes and any other identical serialization.
//
// This is the single attribute-identity check used by addProperty, which
// backs every attribute-creation entry point (SetAttribute, SetAttributeNS,
// SetLiteralAttribute, and SetLiteralAttributeNS). A matching attribute is
// replaced in place; a new one is appended.
func attrMatches(existing *Attribute, qname, nsURI, localName string) bool {
	if existing.URI() == nsURI && existing.LocalName() == localName {
		return true
	}
	return existing.Name() == qname
}

// addProperty inserts or replaces an attribute in the element's property list.
func (n *Element) addProperty(attr *Attribute) {
	p := n.properties
	if p == nil {
		n.properties = attr
		attr.parent = n
		return
	}

	qname := attr.Name()
	nsURI := attr.URI()
	localName := attr.LocalName()

	var last *Attribute
	for ; p != nil; p = p.NextAttribute() {
		if attrMatches(p, qname, nsURI, localName) {
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
	t.parent = attr
	n.addProperty(attr)
	return nil
}

// SetAttributeNS creates or replaces the attribute with the given local name
// and namespace, parsing value for entity references, and returns the element
// for chaining. An existing attribute with the same expanded name (namespace
// URI + local name) or serialized QName is replaced in place. The local name
// must not contain a colon. This is the namespaced analogue of SetAttribute;
// use SetLiteralAttributeNS to store value verbatim without parsing.
func (n *Element) SetAttributeNS(localname, value string, ns *Namespace) (*Element, error) {
	attr, err := n.doc.CreateAttribute(localname, value, ns)
	if err != nil {
		return nil, err
	}

	n.addProperty(attr)
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

// hasAttributeInProperties reports whether p is reachable from this element's
// properties linked list by identity. An *Attribute whose parent is an *Element
// is not guaranteed to live in that element's properties chain: public paths
// such as elem.AddChild(attr) or a generic Replace(attr) can instead place the
// attribute in the normal child list. Property-list splicing must only be used
// when the attribute is genuinely a property; otherwise firstChild/lastChild
// would be left stale.
func (n *Element) hasAttributeInProperties(p *Attribute) bool {
	for attr := n.properties; attr != nil; attr = attr.NextAttribute() {
		if attr == p {
			return true
		}
	}
	return false
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

// Attributes returns a newly allocated slice of the element's attributes in
// property order. The returned slice is a snapshot: appending to or reordering
// it does not affect the element, though the *Attribute elements still point at
// the live attribute nodes. Use ForEachAttribute to avoid the slice allocation.
func (n Element) Attributes() []*Attribute {
	attrs := []*Attribute{}
	for attr := n.properties; attr != nil; attr = attr.NextAttribute() {
		attrs = append(attrs, attr)
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
	for attr := n.properties; attr != nil; attr = attr.NextAttribute() {
		if !fn(attr) {
			return
		}
	}
}

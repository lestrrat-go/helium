package node

import (
	"errors"

	"github.com/lestrrat-go/helium/internal/orderedmap"
)

var ErrDuplicateAttribute = errors.New("duplicate attribute")

type Element struct {
	treeNode
	name   string
	attrs  *orderedmap.Map[string, *Attribute]
	ns     *Namespace
	nsDefs []*Namespace
}

var _ Node = (*Element)(nil)

// NewElement creates a new Element with the given name. Please note
// that elements created this way is an orphan node. You normally want to
// create an element using the Document.CreateElement method, which will
// automatically initialize some data, such as setting the owner document
// for the element.
func NewElement(name string) *Element {
	return &Element{
		name:  name,
		attrs: orderedmap.New[string, *Attribute](),
	}
}

func (Element) Type() NodeType {
	return ElementNodeType
}

func (e *Element) LocalName() string {
	return e.name
}

func (e *Element) AddChild(child Node) error {
	return addChild(e, child)
}

func (e *Element) AddContent(b []byte) error {
	return addContent(e, b)
}

func (e *Element) AddSibling(sibling Node) error {
	return addSibling(e, sibling)
}

func (e *Element) Replace(cur Node) error {
	return replaceNode(e, cur)
}

func (e *Element) SetNextSibling(sibling Node) error {
	return setNextSibling(e, sibling)
}

func (e *Element) SetPrevSibling(sibling Node) error {
	return setPrevSibling(e, sibling)
}

// SetAttribute sets the attribute with the given name. If the name
// of the attribute already exists, it will return an error.
func (e *Element) SetAttribute(name, value string) error {
	attr := e.doc.CreateAttribute(name, value)
	attr.ns = e.ns
	if err := e.attrs.Set(name, attr); err != nil {
		if errors.Is(err, orderedmap.ErrDuplicateEntry) {
			return ErrDuplicateAttribute
		}
	}
	return nil
}

// Attributes populates the given slice with the attributes
// of the element. If the slice is nil, it will create a new slice
// and return it. If the element has no attributes, it will return
// an empty slice.
func (e *Element) Attributes(dst []*Attribute) []*Attribute {
	if dst == nil {
		dst = make([]*Attribute, 0, e.attrs.Len())
	} else {
		dst = dst[:0]
	}
	for _, attr := range e.attrs.Range() {
		dst = append(dst, attr)
	}
	return dst
}

func (e *Element) Name() string {
	if e.ns == nil || e.ns.Prefix() == "" {
		return e.name
	}
	return e.ns.Prefix() + ":" + e.name
}

func (e *Element) Prefix() string {
	if e.ns != nil {
		return e.ns.Prefix()
	}
	return ""
}

func (e *Element) URI() string {
	if e.ns != nil {
		return e.ns.URI()
	}
	return ""
}

// SetNamespace sets the namespace for the element
func (e *Element) SetNamespace(prefixStr, uri string, recursive bool) error {
	// TODO: Implement proper namespace handling
	// For now, just create a simple namespace
	e.ns = NewNamespace(prefixStr, uri)
	return nil
}

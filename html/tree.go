package html

import (
	"github.com/lestrrat-go/helium"
)

// treeBuilder implements SAXHandler and builds a helium DOM tree.
type treeBuilder struct {
	doc *helium.Document
	cur helium.MutableNode // current insertion point
}

func newTreeBuilder() *treeBuilder {
	doc := helium.NewHTMLDocument()
	return &treeBuilder{
		doc: doc,
		cur: doc,
	}
}

func (t *treeBuilder) SetDocumentLocator(loc DocumentLocator) error {
	return nil
}

func (t *treeBuilder) StartDocument() error {
	return nil
}

func (t *treeBuilder) EndDocument() error {
	return nil
}

func (t *treeBuilder) StartElement(name string, attrs []Attribute) error {
	elem := t.doc.CreateElement(name)

	// Use SetLiteralAttribute because the HTML parser has already resolved
	// entities in attribute values. SetAttribute would re-parse them as XML
	// entity references and fail on bare '&' characters.
	// Boolean attributes use SetBooleanAttribute (no children) so the
	// serializer can distinguish them from attrs with empty string values.
	for _, a := range attrs {
		if a.Boolean {
			elem.SetBooleanAttribute(a.Name)
		} else {
			elem.SetLiteralAttribute(a.Name, a.Value)
		}
	}

	if err := t.cur.AddChild(elem); err != nil {
		return err
	}
	t.cur = elem
	return nil
}

func (t *treeBuilder) EndElement(name string) error {
	if t.cur == nil || t.cur == t.doc {
		return nil
	}
	if parent, ok := t.cur.Parent().(helium.MutableNode); ok && parent != nil {
		t.cur = parent
	} else {
		t.cur = t.doc
	}
	return nil
}

func (t *treeBuilder) Characters(ch []byte) error {
	if t.cur == nil || t.cur == t.doc {
		return nil
	}
	return t.cur.AppendText(ch)
}

func (t *treeBuilder) CDataBlock(value []byte) error {
	if t.cur == nil || t.cur == t.doc {
		return nil
	}
	return t.cur.AppendText(value)
}

func (t *treeBuilder) Comment(value []byte) error {
	comment := t.doc.CreateComment(value)
	return t.cur.AddChild(comment)
}

func (t *treeBuilder) InternalSubset(name, externalID, systemID string) error {
	_, err := t.doc.CreateInternalSubset(name, externalID, systemID)
	return err
}

func (t *treeBuilder) ProcessingInstruction(target, data string) error {
	pi := t.doc.CreatePI(target, data)
	return t.cur.AddChild(pi)
}

func (t *treeBuilder) IgnorableWhitespace(ch []byte) error {
	return t.Characters(ch)
}

func (t *treeBuilder) Error(err error) error {
	return nil
}

func (t *treeBuilder) Warning(err error) error {
	return nil
}

package html

import (
	"github.com/lestrrat-go/helium"
)

// treeBuilder implements SAXHandler and builds a helium DOM tree.
type treeBuilder struct {
	doc *helium.Document
	cur helium.Node // current insertion point
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
	elem, err := t.doc.CreateElement(name)
	if err != nil {
		return err
	}

	// Use SetLiteralAttribute because the HTML parser has already resolved
	// entities in attribute values. SetAttribute would re-parse them as XML
	// entity references and fail on bare '&' characters.
	for _, a := range attrs {
		elem.SetLiteralAttribute(a.Name, a.Value)
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
	parent := t.cur.Parent()
	if parent != nil {
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
	return t.cur.AddContent(ch)
}

func (t *treeBuilder) CDataBlock(value []byte) error {
	if t.cur == nil || t.cur == t.doc {
		return nil
	}
	return t.cur.AddContent(value)
}

func (t *treeBuilder) Comment(value []byte) error {
	comment, err := t.doc.CreateComment(value)
	if err != nil {
		return err
	}
	return t.cur.AddChild(comment)
}

func (t *treeBuilder) InternalSubset(name, externalID, systemID string) error {
	_, err := t.doc.CreateInternalSubset(name, externalID, systemID)
	return err
}

func (t *treeBuilder) ProcessingInstruction(target, data string) error {
	pi, err := t.doc.CreatePI(target, data)
	if err != nil {
		return err
	}
	return t.cur.AddChild(pi)
}

func (t *treeBuilder) IgnorableWhitespace(ch []byte) error {
	return t.Characters(ch)
}

func (t *treeBuilder) Error(msg string, args ...interface{}) error {
	return nil
}

func (t *treeBuilder) Warning(msg string, args ...interface{}) error {
	return nil
}

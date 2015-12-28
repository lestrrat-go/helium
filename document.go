package helium

import (
	"bytes"
	"errors"
	"io"
)

func CreateDocument() *Document {
	return NewDocument("1.0", "", StandaloneImplicitNo)
}

func NewDocument(version, encoding string, standalone DocumentStandaloneType) *Document {
	doc := &Document{
		encoding:   encoding,
		standalone: standalone,
		version:    version,
	}
	doc.intSubset, _ = doc.CreateDTD()
	doc.etype = DocumentNode
	doc.name = "(document)"
	return doc
}

func (d Document) XMLString() (string, error) {
	out := bytes.Buffer{}
	if err := d.XML(&out); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (d *Document) XML(out io.Writer) error {
	return (&Dumper{}).DumpDoc(out, d)
}

func (d *Document) AddChild(cur Node) error {
	return addChild(d, cur)
}

func (d *Document) AddContent(b []byte) error {
	return addContent(d, b)
}

func (d *Document) Encoding() string {
	// In order to differentiate between a document with explicit
	// encoding in the XML declaration and one without, the XML dump
	// routine must check for d.encoding == "", and not Encoding()
	if enc := d.encoding; enc != "" {
		return d.encoding
	}
	return "utf8"
}

func (d *Document) Standalone() DocumentStandaloneType {
	return d.standalone
}

func (d *Document) Version() string {
	return d.version
}

func (d *Document) IntSubset() *DTD {
	return d.intSubset
}

func (d *Document) Replace(n Node) {
	panic("d.Replace does not make sense")
}

func (d *Document) SetDocumentElement(root Node) error {
	if d == nil {
		// what are you trying to do?
		return nil
	}

	if root == nil || root.Type() == NamespaceDeclNode {
		return nil
	}

	root.SetParent(d)
	var old Node
	for old = d.firstChild; old != nil; old = old.NextSibling() {
		if old.Type() == ElementNode {
			break
		}
	}

	if old == nil {
		d.AddChild(root)
	} else {
		old.Replace(root)
	}
	return nil
}

func (d *Document) CreateNamespace(prefix, uri string) (*Namespace, error) {
	ns := newNamespace(prefix, uri)
	ns.context = d
	return ns, nil
}

func (d *Document) CreatePI(target, data string) (*ProcessingInstruction, error) {
	return &ProcessingInstruction{
		target: target,
		data:   data,
	}, nil
}

func (d *Document) CreateDTD() (*DTD, error) {
	dtd := newDTD()
	dtd.doc = d
	return dtd, nil
}

func (d *Document) CreateElement(name string) (*Element, error) {
	e := newElement(name)
	e.doc = d
	return e, nil
}

func (d *Document) CreateText(value []byte) (*Text, error) {
	e := newText(value)
	e.doc = d
	return e, nil
}

func (d *Document) CreateComment(value []byte) (*Comment, error) {
	e := newComment(value)
	e.doc = d
	return e, nil
}

func (d *Document) CreateElementContent(name string, etype ElementContentType) (*ElementContent, error) {
	e, err := newElementContent(name, etype)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (d *Document) GetEntity(name string) (*Entity, bool) {
	if ints := d.intSubset; ints != nil {
		return ints.LookupEntity(name)
	}

	if exts := d.extSubset; exts != nil {
		return exts.LookupEntity(name)
	}

	return nil, false
}

func (d *Document) GetParameterEntity(name string) (*Entity, bool) {
	if ints := d.intSubset; ints != nil {
		return ints.LookupParameterEntity(name)
	}

	if exts := d.extSubset; exts != nil {
		return exts.LookupParameterEntity(name)
	}

	return nil, false
}

func (d *Document) IsMixedElement(name string) (bool, error) {
	edecl, ok := d.intSubset.GetElementDesc(name)
	if !ok {
		return false, errors.New("element declaration not found")
	}

	switch edecl.decltype {
	case UndefinedElementType:
		return false, errors.New("element declaration not found")
	case ElementElementType:
		return false, nil
	case EmptyElementType, AnyElementType, MixedElementType:
		/*
		 * return 1 for EMPTY since we want VC error to pop up
		 * on <empty>     </empty> for example
		 */
		return true, nil
	}
	return true, nil
}

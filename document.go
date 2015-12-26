package helium

import "errors"

func NewDocument(version, encoding string, standalone DocumentStandaloneType) *Document {
	doc := &Document{
		encoding:   encoding,
		standalone: standalone,
		version:    version,
	}
	doc.intSubset, _ = doc.CreateDTD()
	return doc
}

func (d *Document) Encoding() string {
	return d.encoding
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

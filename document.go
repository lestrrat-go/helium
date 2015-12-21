package helium

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

func (d *Document) IntSubset() Node {
	return d.intSubset
}

func (d *Document) CreatePI(target, data string) (*ProcessingInstruction, error) {
	return &ProcessingInstruction{
		target: target,
		data:   data,
	}, nil
}

func (d *Document) CreateDTD() (*DTD, error) {
	return &DTD{
		node:      node{},
		entities:  map[string]*Entity{},
		pentities: map[string]*Entity{},
	}, nil
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

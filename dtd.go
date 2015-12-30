package helium

import "errors"

func newDTD() *DTD {
	dtd := &DTD{
		attributes: map[string]*AttributeDecl{},
		elements:  map[string]ElementDecl{},
		entities:  map[string]*Entity{},
		pentities: map[string]*Entity{},
	}
	dtd.etype = DTDNode
	return dtd
}

func (dtd *DTD) RegisterEntity(name string, typ EntityType, publicID, systemID, content string) (*Entity, error) {
	var table map[string]*Entity
	switch typ {
	case InternalGeneralEntity, ExternalGeneralParsedEntity, ExternalGeneralUnparsedEntity:
		table = dtd.entities
	case InternalParameterEntity, ExternalParameterEntity:
		table = dtd.pentities
	case InternalPredefinedEntity:
		return nil, errors.New("cannot register a predefined entity")
	}

	ent := newEntity(name, typ, publicID, systemID, content, "")
	ent.doc = dtd.doc
	table[name] = ent
	return ent, nil
}

func (dtd *DTD) LookupAttribute(name, prefix, elem string) (*AttributeDecl, bool) {
	key := name + ":" + prefix + ":" + elem
	decl, ok := dtd.attributes[key]
	if !ok {
		return nil, false
	}
	return decl, ok
}

func (dtd *DTD) RegisterAttribute(attr *AttributeDecl) error {
	// TODO maybe this shouldn't be normalized, check later
	key := attr.name + ":" + attr.prefix + ":" + attr.elem
	_, ok := dtd.attributes[key]
	if ok {
		return errors.New("duplicate attribute declared")
	}
	dtd.attributes[key] = attr
	return nil
}

func (dtd *DTD) LookupEntity(name string) (*Entity, bool) {
	ret, ok := dtd.entities[name]
	return ret, ok
}

func (dtd *DTD) LookupParameterEntity(name string) (*Entity, bool) {
	ret, ok := dtd.pentities[name]
	return ret, ok
}

func (dtd *DTD) GetElementDesc(name string) (*ElementDecl, bool) {
	ret, ok := dtd.elements[name]
	return &ret, ok
}

func (dtd *DTD) AddChild(cur Node) error {
	return addChild(dtd, cur)
}

func (dtd *DTD) AddContent(b []byte) error {
	return addContent(dtd, b)
}

func (dtd *DTD) AddSibling(cur Node) error {
	return addSibling(dtd, cur)
}

func (dtd *DTD) Replace(cur Node) {
	replaceNode(dtd, cur)
}

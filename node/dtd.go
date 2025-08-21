package node

// DTD RegisterAttribute method
func (dtd *DTD) RegisterAttribute(attr *AttributeDecl) error {
	if dtd.attributes == nil {
		dtd.attributes = make(map[string]*AttributeDecl)
	}
	key := attr.elem + ":" + attr.LocalName()
	dtd.attributes[key] = attr
	return nil
}

// ElementDecl LocalName method
func (e *ElementDecl) LocalName() string {
	return e.name
}

// DTD AddChild method
func (dtd *DTD) AddChild(cur Node) error {
	return addChild(dtd, cur)
}

// DTD AddContent method
func (dtd *DTD) AddContent(b []byte) error {
	return addContent(dtd, b)
}

// DTD AddSibling method
func (dtd *DTD) AddSibling(cur Node) error {
	return addSibling(dtd, cur)
}

// DTD Replace method
func (dtd *DTD) Replace(cur Node) error {
	return replaceNode(dtd, cur)
}

// DTD Type method
func (dtd *DTD) Type() NodeType {
	return DTDNodeType
}

// DTD LocalName method
func (dtd *DTD) LocalName() string {
	return "#dtd"
}

// DTD SetNextSibling method
func (dtd *DTD) SetNextSibling(sibling Node) error {
	return setNextSibling(dtd, sibling)
}

// DTD SetPrevSibling method
func (dtd *DTD) SetPrevSibling(sibling Node) error {
	return setPrevSibling(dtd, sibling)
}

// DTD Free method (stub)
func (dtd *DTD) Free() {
	// TODO: Implement proper DTD cleanup
}

// DTD AddElementDecl adds an element declaration
func (dtd *DTD) AddElementDecl(decl *ElementDecl) error {
	if dtd.elements == nil {
		dtd.elements = make(map[string]*ElementDecl)
	}
	dtd.elements[decl.LocalName()] = decl
	return nil
}

// DTD AddEntity adds an entity
func (dtd *DTD) AddEntity(entity *Entity) error {
	if dtd.entities == nil {
		dtd.entities = make(map[string]*Entity)
	}
	dtd.entities[entity.LocalName()] = entity
	return nil
}

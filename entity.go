package helium

import "errors"

func resolvePredefinedEntity(name string) (*Entity, error) {
	switch name {
	case "lt":
		return EntityLT, nil
	case "gt":
		return EntityGT, nil
	case "amp":
		return EntityAmpersand, nil
	case "apos":
		return EntityApostrophe, nil
	case "quot":
		return EntityQuote, nil
	default:
		return nil, errors.New("entity not found")
	}
}

func newEntity(name string, typ EntityType, publicID, systemID, notation, orig string) *Entity {
	e := &Entity{
		content:    notation,
		entityType: typ,
		externalID: publicID,
		systemID:   systemID,
		orig:       orig,
	}
	e.etype = EntityNode
	e.name = name
	return e
}

func (e *Entity) SetOrig(s string) {
	e.orig = s
}

func (e *Entity) EntityType() int {
	return int(e.entityType)
}

func (e *Entity) Content() []byte {
	return []byte(e.content)
}

func (e *Entity) AddChild(cur Node) error {
	return addChild(e, cur)
}

func (e *Entity) AddContent(b []byte) error {
	return addContent(e, b)
}

func (e *Entity) AddSibling(cur Node) error {
	return addSibling(e, cur)
}

func (e *Entity) Replace(cur Node) {
	replaceNode(e, cur)
}

func (n *Entity) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

package helium

func resolvePredefinedEntity(name string) *Entity {
	switch name {
	case "lt":
		return EntityLT
	case "gt":
		return EntityGT
	case "amp":
		return EntityAmpersand
	case "apos":
		return EntityApostrophe
	case "quot":
		return EntityQuote
	default:
		return nil
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
	e.name = name
	return e
}

func (e *Entity) SetOrig(s string) {
	e.orig = s
}

func (e *Entity) EntityType() int {
	return int(e.entityType)
}

func (e *Entity) Content() string {
	return e.content
}

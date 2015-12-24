package helium

func resolvePredefinedEntity(name string) *Entity {
	switch name {
	case "lt":
		return &EntityLT
	case "gt":
		return &EntityGT
	case "amp":
		return &EntityAmpersand
	case "apos":
		return &EntityApostrophe
	case "quot":
		return &EntityQuote
	default:
		return nil
	}
}

func NewEntity(orig string, typ EntityType, publicID, systemID, notation string) *Entity {
	return &Entity{
		orig:       orig,
		content:    notation,
		entityType: typ,
		externalID: publicID,
		systemID:   systemID,
	}
}

func (e *Entity) EntityType() EntityType {
	return e.entityType
}

func (e *Entity) Content() string {
	return e.content
}

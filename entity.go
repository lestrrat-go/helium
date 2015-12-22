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

func (e *Entity) EntityType() EntityType {
	return e.entityType
}

func (e *Entity) Content() string {
	return e.content
}

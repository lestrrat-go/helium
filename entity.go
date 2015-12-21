package helium

func resolvePredefinedEntity(name string) *Entity {
	switch name {
	case "lt":
		return &EntityLT
	case "apos":
		return &EntityApostrophe
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

package helium

func newDTD() *DTD {
	return &DTD {
		elements:  map[string]ElementDecl{},
		entities:  map[string]Entity{},
		pentities: map[string]Entity{},
	}
}

func (dtd *DTD) LookupEntity(name string) (*Entity, bool) {
	ret, ok := dtd.entities[name]
	return &ret, ok
}

func (dtd *DTD) LookupParameterEntity(name string) (*Entity, bool) {
	ret, ok := dtd.pentities[name]
	return &ret, ok
}

func (dtd *DTD) GetElementDesc(name string) (*ElementDecl, bool) {
	ret, ok := dtd.elements[name]
	return &ret, ok
}

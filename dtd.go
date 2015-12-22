package helium

func (dtd *DTD) LookupEntity(name string) (*Entity, bool) {
	ret, ok := dtd.entities[name]
	return &ret, ok
}

func (dtd *DTD) LookupParameterEntity(name string) (*Entity, bool) {
	ret, ok := dtd.pentities[name]
	return &ret, ok
}

func (dtd *DTD) GetElementDesc(name string) (*Element, bool) {
	ret, ok := dtd.elements[name]
	return &ret, ok
}

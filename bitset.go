package helium

func (p *ParseOption) Set(n ParseOption) {
	*p = *p | n
}

func (p ParseOption) IsSet(n ParseOption) bool {
	return p&n != 0
}

func (p *LoadSubsetOption) Set(n LoadSubsetOption) {
	*p = *p | n
}

func (p LoadSubsetOption) IsSet(n LoadSubsetOption) bool {
	return p&n != 0
}

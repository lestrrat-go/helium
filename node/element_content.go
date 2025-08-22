package node

func newElementContent(name, prefix string, ctype ElementContentType, occur ElementContentOccur) *ElementContent {
	return &ElementContent{
		name:   name,
		prefix: prefix,
		ctype:  ctype,
		occur:  occur,
	}
}

func (ec *ElementContent) SetChildren(c1, c2 *ElementContent) {
	ec.c1 = c1
	ec.c2 = c2
}

func (ec *ElementContent) SetParent(p *ElementContent) {
	ec.parent = p
}

func (ec *ElementContent) SetOccur(occur ElementContentOccur) {
	ec.occur = occur
}

func (ec *ElementContent) GetOccur() ElementContentOccur {
	return ec.occur
}

func (ec *ElementContent) GetC1() *ElementContent {
	return ec.c1
}

func (ec *ElementContent) SetC1(c1 *ElementContent) {
	ec.c1 = c1
}

func (ec *ElementContent) GetC2() *ElementContent {
	return ec.c2
}

func (ec *ElementContent) SetC2(c2 *ElementContent) {
	ec.c2 = c2
}

func (ec *ElementContent) GetParent() *ElementContent {
	return ec.parent
}

func (ec *ElementContent) GetType() ElementContentType {
	return ec.ctype
}

func (ec *ElementContent) GetName() string {
	return ec.name
}

// Legacy field accessors for backward compatibility with helium package code
func (ec *ElementContent) GetCoccur() ElementContentOccur {
	return ec.occur
}

func (ec *ElementContent) SetCoccur(occur ElementContentOccur) {
	ec.occur = occur
}

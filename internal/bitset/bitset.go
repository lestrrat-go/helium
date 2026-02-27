package bitset

type Field interface {
	~int
}

func Set[T Field](p *T, n T) {
	*p = *p | n
}

func IsSet[T Field](p T, n T) bool {
	return p&n != 0
}

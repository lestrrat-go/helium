package stack

type AnyItem interface{}
type SimpleStack []AnyItem

func (s *SimpleStack) Push(i AnyItem) {
	*s = append(*s, i)
}

func (s *SimpleStack) Pop(n ...int) {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	stackPop(s, nn)
}

func (s *SimpleStack) Realloc() {
	*s = append(SimpleStack(nil), *s...)
}

func (s *SimpleStack) PopLast() {
	if s.Len() <= 0 {
		return
	}
	*s = (*s)[:s.Len()-1]
}

func (s SimpleStack) Peek(n int) []AnyItem {
	if l := s.Len(); l > n {
		return s[l-n : l]
	}
	return s
}

func (s SimpleStack) Len() int {
	return len(s)
}

func (s SimpleStack) Cap() int {
	return cap(s)
}

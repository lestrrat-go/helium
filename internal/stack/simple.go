package stack

type Stack[T any] []T

func (s *Stack[T]) Push(i T) {
	*s = append(*s, i)
}

func (s *Stack[T]) Pop(n ...int) {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	stackPop(s, nn)
}

func (s *Stack[T]) Realloc() {
	*s = append(Stack[T](nil), *s...)
}

func (s *Stack[T]) PopLast() {
	if s.Len() <= 0 {
		return
	}
	*s = (*s)[:s.Len()-1]
}

func (s Stack[T]) Peek(n int) []T {
	if l := s.Len(); l > n {
		return s[l-n : l]
	}
	return s
}

func (s Stack[T]) Len() int {
	return len(s)
}

func (s Stack[T]) Cap() int {
	return cap(s)
}

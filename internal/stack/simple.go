package stack

// Stack is a LIFO stack whose top is the last element of the slice.
type Stack[T any] []T

// Push adds i to the top of the stack.
func (s *Stack[T]) Push(i T) {
	*s = append(*s, i)
}

// Pop removes the top n entries (one when n is omitted), or every entry when
// the stack holds fewer. It keeps the backing array; see the package comment.
func (s *Stack[T]) Pop(n ...int) {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	*s = truncate(*s, nn)
}

// Peek returns the top n entries, oldest first, or the whole stack when it
// holds n entries or fewer. The result aliases the stack.
func (s Stack[T]) Peek(n int) []T {
	if l := s.Len(); l > n {
		return s[l-n : l]
	}
	return s
}

// Len returns the number of entries on the stack.
func (s Stack[T]) Len() int {
	return len(s)
}

// Cap returns the capacity of the stack's backing array.
func (s Stack[T]) Cap() int {
	return cap(s)
}

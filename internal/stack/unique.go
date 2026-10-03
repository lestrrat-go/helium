package stack

import "errors"

var ErrDuplicateItem = errors.New("item already exists")

type Keyed interface {
	Key() string
}

// KeyedStack is a LIFO stack of keyed entries, searched from the top by key.
type KeyedStack[T Keyed] []T

// Push adds i to the top of the stack, unless an entry with the same key is
// already on it, in which case it returns ErrDuplicateItem.
func (s *KeyedStack[T]) Push(i T) error {
	if _, ok := s.Lookup(i.Key()); ok {
		return ErrDuplicateItem
	}
	*s = append(*s, i)
	return nil
}

// Pop removes the top n entries (one when n is omitted), or every entry when
// the stack holds fewer. It keeps the backing array; see the package comment.
func (s *KeyedStack[T]) Pop(n ...int) {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	*s = truncate(*s, nn)
}

// Len returns the number of entries on the stack.
func (s KeyedStack[T]) Len() int {
	return len(s)
}

// Cap returns the capacity of the stack's backing array.
func (s KeyedStack[T]) Cap() int {
	return cap(s)
}

// Lookup returns the topmost entry whose key is key.
func (s KeyedStack[T]) Lookup(key string) (T, bool) {
	for i := s.Len() - 1; i >= 0; i -= 1 {
		if s[i].Key() == key {
			return s[i], true
		}
	}
	var zero T
	return zero, false
}

// Peek returns the top n entries, oldest first, or the whole stack when it
// holds n entries or fewer. The result aliases the stack.
func (s KeyedStack[T]) Peek(n int) []T {
	if l := s.Len(); l > n {
		return s[l-n : l]
	}
	return s
}

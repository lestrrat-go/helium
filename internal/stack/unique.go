package stack

import "errors"

var ErrDuplicateItem = errors.New("item already exists")

type Keyed interface {
	Key() string
}

type KeyedStack[T Keyed] []T

func (s *KeyedStack[T]) Push(i T) error {
	if _, ok := s.Lookup(i.Key()); ok {
		return ErrDuplicateItem
	}
	*s = append(*s, i)
	return nil
}

func (s *KeyedStack[T]) Pop(n ...int) {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	stackPop(s, nn)
}

func (s *KeyedStack[T]) Realloc() {
	*s = append(KeyedStack[T](nil), *s...)
}

func (s *KeyedStack[T]) PopLast() {
	if s.Len() <= 0 {
		return
	}
	*s = (*s)[:s.Len()-1]
}

func (s KeyedStack[T]) Len() int {
	return len(s)
}

func (s KeyedStack[T]) Cap() int {
	return cap(s)
}

func (s KeyedStack[T]) Lookup(key string) (T, bool) {
	for i := s.Len() - 1; i >= 0; i -= 1 {
		if s[i].Key() == key {
			return s[i], true
		}
	}
	var zero T
	return zero, false
}

func (s KeyedStack[T]) Peek(n int) []T {
	if l := s.Len(); l > n {
		return s[l-n : l]
	}
	return s
}

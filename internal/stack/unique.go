package stack

import "errors"

var ErrDuplicateItem = errors.New("item already exists")

type LookupItem interface {
	Key() string
}

type UniqueStack []LookupItem

func (s *UniqueStack) Push(i LookupItem) error {
	if s.Lookup(i.Key()) != NilItem {
		return ErrDuplicateItem
	}
	*s = append(*s, i)
	return nil
}

func (s *UniqueStack) Pop(n ...int) {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	stackPop(s, nn)
}

func (s *UniqueStack) Realloc() {
	*s = append(UniqueStack(nil), *s...)
}

func (s *UniqueStack) PopLast() {
	if s.Len() <= 0 {
		return
	}
	*s = (*s)[:s.Len()-1]
}

func (s UniqueStack) Len() int {
	return len(s)
}

func (s UniqueStack) Cap() int {
	return cap(s)
}

func (s UniqueStack) Lookup(key string) LookupItem {
	for i := s.Len() - 1; i >= 0; i -= 1 {
		if s[i].Key() == key {
			return s[i]
		}
	}
	return NilItem
}

func (s UniqueStack) Peek(n int) []LookupItem {
	if l := s.Len(); l > n {
		return s[l-n:l]
	}
	return s
}

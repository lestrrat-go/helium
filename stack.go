package helium

import "github.com/lestrrat-go/helium/internal/stack"

type nodeStack struct {
	stack.SimpleStack
}

type inputStack struct {
	stack.SimpleStack
}

type nsStack struct {
	stack.UniqueStack
}

type nsStackItem struct {
	prefix string
	href   string
}

// Appease the sax.Namespace interface
func (i nsStackItem) Prefix() string {
	return i.prefix
}

// Appease the sax.Namespace interface
func (i nsStackItem) URI() string {
	return i.href
}

func (i nsStackItem) Key() string {
	return i.prefix
}

func (s *nsStack) Push(prefix, uri string) {
	_ = s.UniqueStack.Push(nsStackItem{prefix: prefix, href: uri})
}

func (s *nsStack) Lookup(prefix string) string {
	item := s.UniqueStack.Lookup(prefix)
	if item == stack.NilItem {
		return ""
	}
	return item.(nsStackItem).href
}

func (s *nodeStack) Push(e *Element) {
	s.SimpleStack.Push(stack.AnyItem(e))
}

func (s *nodeStack) Pop() *Element {
	defer s.SimpleStack.Pop()
	if e := s.PeekOne(); e != nil {
		return e
	}
	return nil
}

func (s *nodeStack) PeekOne() *Element {
	l := s.SimpleStack.Peek(1) // nolint:staticcheck
	if len(l) != 1 {
		return nil
	}
	return l[0].(*Element)
}

// the reason we're using interface{} here is that we may have to
// push a ByteCursor or a RuneCursor, and they don't share
// a common API
func (s *inputStack) Push(c interface{}) {
	s.SimpleStack.Push(stack.AnyItem(c))
}

func (s *inputStack) Pop() interface{} {
	defer s.SimpleStack.Pop() // nolint:staticcheck
	if e := s.PeekOne(); e != nil {
		return e
	}
	return nil
}

func (s *inputStack) PeekOne() interface{} {
	l := s.SimpleStack.Peek(1) // nolint:staticcheck
	if len(l) != 1 {
		return nil
	}
	return l[0].(interface{})
}

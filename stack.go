package helium

import "github.com/lestrrat-go/helium/internal/stack"

type nodeStack struct {
	stack.Stack[*Element]
}

type inputStack struct {
	stack.Stack[any]
}

type nsStack struct {
	stack.KeyedStack[nsStackItem]
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
	// Force-append: namespace prefixes may be redeclared on child elements
	// (shadowing the parent's binding), so we must allow duplicate keys.
	// KeyedStack.Lookup searches from the end, giving correct shadowing.
	s.KeyedStack = append(s.KeyedStack, nsStackItem{prefix: prefix, href: uri})
}

func (s *nsStack) Lookup(prefix string) string {
	item, ok := s.KeyedStack.Lookup(prefix)
	if !ok {
		return ""
	}
	return item.href
}

func (s *nodeStack) Push(e *Element) {
	s.Stack.Push(e)
}

func (s *nodeStack) Pop() *Element {
	defer s.Stack.Pop()
	if e := s.PeekOne(); e != nil {
		return e
	}
	return nil
}

func (s *nodeStack) PeekOne() *Element {
	l := s.Stack.Peek(1)
	if len(l) != 1 {
		return nil
	}
	return l[0]
}

// the reason we're using any here is that we may have to
// push a ByteCursor or a RuneCursor, and they don't share
// a common API
func (s *inputStack) Push(c any) {
	s.Stack.Push(c)
}

func (s *inputStack) Pop() any {
	defer s.Stack.Pop()
	if e := s.PeekOne(); e != nil {
		return e
	}
	return nil
}

func (s *inputStack) PeekOne() any {
	l := s.Stack.Peek(1)
	if len(l) != 1 {
		return nil
	}
	return l[0]
}

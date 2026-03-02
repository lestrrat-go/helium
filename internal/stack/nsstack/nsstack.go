package nsstack

import "github.com/lestrrat-go/helium/internal/stack"

type Item struct {
	prefix string
	href   string
}

// Appease the sax.Namespace interface
func (i Item) Prefix() string {
	return i.prefix
}

// Appease the sax.Namespace interface
func (i Item) URI() string {
	return i.href
}

func (i Item) Key() string {
	return i.prefix
}

type Stack struct {
	stack.KeyedStack[Item]
}

func New() Stack {
	return Stack{}
}

func (s *Stack) Push(prefix, uri string) {
	_ = s.KeyedStack.Push(Item{prefix: prefix, href: uri})
}

func (s *Stack) Lookup(prefix string) string {
	item, ok := s.KeyedStack.Lookup(prefix)
	if !ok {
		return ""
	}
	return item.href
}

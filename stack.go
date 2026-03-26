package helium

import "github.com/lestrrat-go/helium/internal/stack"

// nodeEntry is a lightweight record for the parser's element stack.
// It stores only the data needed for end-tag matching and SAX callbacks,
// avoiding a full *Element allocation per start tag.
type nodeEntry struct {
	local  string
	prefix string
	uri    string
	qname  string
}

func (e *nodeEntry) Name() string {
	return e.qname
}

func (e *nodeEntry) LocalName() string {
	return e.local
}

func (e *nodeEntry) Prefix() string {
	return e.prefix
}

func (e *nodeEntry) URI() string {
	return e.uri
}

type nodeStack struct {
	stack.Stack[nodeEntry]
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

// LookupInTopN searches only the top n entries of the stack for prefix.
// This is used to detect duplicate namespace declarations on the same
// element without being confused by ancestor bindings (which are valid
// prefix shadowing, not duplicates).
func (s *nsStack) LookupInTopN(prefix string, n int) string {
	entries := s.Peek(n)
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].prefix == prefix {
			return entries[i].href
		}
	}
	return ""
}

func (s *nodeStack) Push(e nodeEntry) {
	s.Stack.Push(e)
}

func (s *nodeStack) Pop() *nodeEntry {
	l := s.Peek(1)
	if len(l) != 1 {
		return nil
	}
	e := &l[0]
	s.Stack.Pop()
	return e
}

func (s *nodeStack) PeekOne() *nodeEntry {
	l := s.Peek(1)
	if len(l) != 1 {
		return nil
	}
	return &l[0]
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
	l := s.Peek(1)
	if len(l) != 1 {
		return nil
	}
	return l[0]
}

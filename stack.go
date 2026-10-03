package helium

import (
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/stack"
)

// nodeEntry is a lightweight record for the parser's element stack.
// It stores only the data needed for end-tag matching and SAX callbacks,
// avoiding a full *Element allocation per start tag.
type nodeEntry struct {
	local  string
	prefix string
	uri    string
	qname  string
	// synthetic marks the internal pseudo-root that wraps entity replacement
	// text / a parsed fragment (see pseudoRootName). Its name is chosen by the
	// parser, not the document, so whitespace classification must NOT consult a
	// DTD element declaration that happens to match that synthetic name — see
	// areBlanksBytes / whitespaceContextIgnorable.
	synthetic bool
	// declType caches the element's DTD content-model type for whitespace
	// classification (parserCtx.nodeDeclType): declState is declUnknown until
	// the first lookup, then declMissing when neither subset declares the
	// element, or declKnown with declType holding the declared type. Both DTD
	// subsets are complete before the root element opens, so the answer cannot
	// change while the entry is on the stack.
	declType  enum.ElementType
	declState uint8
}

const (
	declUnknown uint8 = iota
	declMissing
	declKnown
)

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

func (s *nodeStack) Push(e nodeEntry) {
	s.Stack.Push(e)
}

// Pop removes the top entry. It truncates the slice directly: the parser pops
// once per end tag, and the backing array keeps its capacity so the next
// descent to the same depth does not reallocate. The popped slot is cleared so
// its strings do not stay reachable from the stack.
func (s *nodeStack) Pop() {
	n := len(s.Stack)
	if n == 0 {
		return
	}
	s.Stack[n-1] = nodeEntry{}
	s.Stack = s.Stack[:n-1]
}

func (s *nodeStack) PeekOne() *nodeEntry {
	l := s.Peek(1)
	if len(l) != 1 {
		return nil
	}
	return &l[0]
}

// Push adds an input cursor: a *strcursor.ByteCursor for the raw document
// bytes, parameter-entity text, and the external subset, or the
// *strcursor.UTF8Cursor that switchEncoding pushes over the decoded document.
func (s *inputStack) Push(c any) {
	s.Stack.Push(c)
}

// Pop removes the top input and returns it, or returns nil when the stack is
// empty.
func (s *inputStack) Pop() any {
	e := s.PeekOne()
	s.Stack.Pop()
	return e
}

func (s *inputStack) PeekOne() any {
	l := s.Peek(1)
	if len(l) != 1 {
		return nil
	}
	return l[0]
}

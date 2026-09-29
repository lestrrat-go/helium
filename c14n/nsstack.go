package c14n

import "iter"

// bindingStack is a scoped prefix→URI map. It keeps one live map plus an undo
// log, so push is free, set records the previous value once per frame, pop
// replays the log back to the frame's mark, and lookup is a single map read.
//
// The canonicalizer uses two instances: one tracks what visible ancestors have
// rendered, the other the in-scope namespace bindings of the element being
// walked.
type bindingStack struct {
	live  map[string]bindingEntry
	undo  []binding
	marks []int
}

// bindingEntry is one live binding and the frame depth that set it.
type bindingEntry struct {
	uri   string
	frame int
}

// binding is one undo-log record: the prefix set in a frame and the entry it
// replaced (had is false when the prefix was unbound).
type binding struct {
	prefix string
	prev   bindingEntry
	had    bool
}

func newBindingStack() *bindingStack {
	return &bindingStack{live: make(map[string]bindingEntry)}
}

// push opens a new frame.
func (s *bindingStack) push() {
	s.marks = append(s.marks, len(s.undo))
}

// pop closes the current frame, restoring every binding it changed.
func (s *bindingStack) pop() {
	if len(s.marks) == 0 {
		return
	}
	mark := s.marks[len(s.marks)-1]
	s.marks = s.marks[:len(s.marks)-1]
	for i := len(s.undo) - 1; i >= mark; i-- {
		b := s.undo[i]
		if b.had {
			s.live[b.prefix] = b.prev
		} else {
			delete(s.live, b.prefix)
		}
	}
	clear(s.undo[mark:])
	s.undo = s.undo[:mark]
}

// set binds prefix to uri in the current frame. A prefix set twice in one frame
// keeps a single undo record, so frameDelta yields it once.
func (s *bindingStack) set(prefix, uri string) {
	frame := len(s.marks)
	prev, had := s.live[prefix]
	if had && prev.frame == frame {
		s.live[prefix] = bindingEntry{uri: uri, frame: frame}
		return
	}
	s.undo = append(s.undo, binding{prefix: prefix, prev: prev, had: had})
	s.live[prefix] = bindingEntry{uri: uri, frame: frame}
}

// lookup returns the URI currently bound to prefix.
func (s *bindingStack) lookup(prefix string) (string, bool) {
	e, ok := s.live[prefix]
	return e.uri, ok
}

// all yields every live binding, in map order.
func (s *bindingStack) all() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for prefix, e := range s.live {
			if !yield(prefix, e.uri) {
				return
			}
		}
	}
}

// frameDelta yields each prefix set in the current frame once, with its current
// URI, in the order the prefixes were first set. A prefix set to the same URI
// its enclosing frame had is still yielded.
func (s *bindingStack) frameDelta() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		mark := 0
		if len(s.marks) > 0 {
			mark = s.marks[len(s.marks)-1]
		}
		for _, b := range s.undo[mark:] {
			if !yield(b.prefix, s.live[b.prefix].uri) {
				return
			}
		}
	}
}

// needsOutput reports whether rendering (prefix, uri) changes what this stack
// records: the prefix was never bound, or is bound to a different URI. An
// unbound default namespace with an empty URI needs no output, since the empty
// default namespace is implicit.
func (s *bindingStack) needsOutput(prefix, uri string) bool {
	existingURI, found := s.lookup(prefix)
	if !found {
		return uri != ""
	}
	return existingURI != uri
}

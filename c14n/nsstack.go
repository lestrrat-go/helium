package c14n

// visibleNSStack tracks which (prefix, URI) pairs have been rendered
// by visible ancestors, so we can determine which namespace declarations
// need to be output on the current element.
type visibleNSStack struct {
	frames []nsFrame
}

type nsFrame struct {
	// rendered maps prefix → URI for namespaces rendered in this frame
	rendered map[string]string
}

func newVisibleNSStack() *visibleNSStack {
	return &visibleNSStack{
		frames: []nsFrame{{rendered: make(map[string]string)}},
	}
}

// save pushes a new frame onto the stack.
func (s *visibleNSStack) save() {
	s.frames = append(s.frames, nsFrame{rendered: make(map[string]string)})
}

// restore pops the top frame from the stack.
func (s *visibleNSStack) restore() {
	if len(s.frames) > 1 {
		s.frames = s.frames[:len(s.frames)-1]
	}
}

// lookup checks if a (prefix, URI) pair has already been rendered
// by an ancestor frame.
func (s *visibleNSStack) lookup(prefix string) (string, bool) {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if uri, ok := s.frames[i].rendered[prefix]; ok {
			return uri, true
		}
	}
	return "", false
}

// add records that a (prefix, URI) pair has been rendered in the current frame.
func (s *visibleNSStack) add(prefix, uri string) {
	s.frames[len(s.frames)-1].rendered[prefix] = uri
}

// needsOutput returns true if this (prefix, URI) pair hasn't been rendered
// yet or has been rendered with a different URI.
func (s *visibleNSStack) needsOutput(prefix, uri string) bool {
	existingURI, found := s.lookup(prefix)
	if !found {
		// Never rendered this prefix. For the default namespace with empty URI,
		// we suppress it (C14N: default namespace with empty URI is implicit).
		return uri != ""
	}
	return existingURI != uri
}

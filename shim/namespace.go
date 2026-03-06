package shim

// nsEntry represents a single namespace binding in scope.
type nsEntry struct {
	prefix string
	uri    string
}

// nsScope represents one level of namespace bindings (one element's worth).
type nsScope struct {
	bindings []nsEntry
}

// nsStack tracks namespace bindings for the encoder, matching
// encoding/xml behavior where each StartElement pushes a new scope.
type nsStack struct {
	scopes    []nsScope
	nextID    int // for auto-generating prefixes
}

func (s *nsStack) push() {
	s.scopes = append(s.scopes, nsScope{})
}

func (s *nsStack) pop() {
	if len(s.scopes) > 0 {
		s.scopes = s.scopes[:len(s.scopes)-1]
	}
}

func (s *nsStack) addBinding(prefix, uri string) {
	if len(s.scopes) == 0 {
		s.scopes = append(s.scopes, nsScope{})
	}
	top := &s.scopes[len(s.scopes)-1]
	top.bindings = append(top.bindings, nsEntry{prefix: prefix, uri: uri})
}

// resolve returns the prefix bound to the given URI, or "" if unbound.
func (s *nsStack) resolve(uri string) (string, bool) {
	for i := len(s.scopes) - 1; i >= 0; i-- {
		for _, b := range s.scopes[i].bindings {
			if b.uri == uri {
				return b.prefix, true
			}
		}
	}
	return "", false
}

// allocPrefix generates a unique prefix for an unbound namespace URI.
func (s *nsStack) allocPrefix() string {
	s.nextID++
	return "ns" + itoa(s.nextID)
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}

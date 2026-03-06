package shim

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const xmlNSURI = "http://www.w3.org/XML/1998/namespace"

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
	scopes []nsScope
	// attrNS maps prefix → URI for allocated attribute prefixes.
	attrNS map[string]string
	// attrPrefix maps URI → prefix for allocated attribute prefixes.
	attrPrefix map[string]string
	seq        int // collision suffix counter
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

// allocPrefix derives a prefix for the given namespace URI, matching
// encoding/xml's createAttrPrefix algorithm. It returns the prefix
// and whether an xmlns declaration needs to be emitted.
func (s *nsStack) allocPrefix(uri string) (string, bool) {
	if p, ok := s.attrPrefix[uri]; ok {
		return p, false
	}

	if uri == xmlNSURI {
		return "xml", false
	}

	if s.attrPrefix == nil {
		s.attrPrefix = make(map[string]string)
		s.attrNS = make(map[string]string)
	}

	// Derive candidate from URI: last path segment
	candidate := strings.TrimRight(uri, "/")
	if i := strings.LastIndex(candidate, "/"); i >= 0 {
		candidate = candidate[i+1:]
	}
	if candidate == "" || !isXMLName(candidate) || strings.Contains(candidate, ":") {
		candidate = "_"
	}

	// Reserved: anything starting with xml (case-insensitive)
	if len(candidate) >= 3 && strings.EqualFold(candidate[:3], "xml") {
		candidate = "_" + candidate
	}

	// Resolve collisions
	prefix := candidate
	if s.attrNS[prefix] != "" {
		for s.seq++; ; s.seq++ {
			id := candidate + "_" + strconv.Itoa(s.seq)
			if s.attrNS[id] == "" {
				prefix = id
				break
			}
		}
	}

	s.attrPrefix[uri] = prefix
	s.attrNS[prefix] = uri

	return prefix, true
}

// isXMLName checks whether s is a valid XML Name, matching the behavior
// of encoding/xml's internal isName function.
func isXMLName(s string) bool {
	if len(s) == 0 {
		return false
	}
	c, n := utf8.DecodeRuneInString(s)
	if c == utf8.RuneError && n == 1 {
		return false
	}
	if !unicode.Is(xmlFirst, c) {
		return false
	}
	for n < len(s) {
		s = s[n:]
		c, n = utf8.DecodeRuneInString(s)
		if c == utf8.RuneError && n == 1 {
			return false
		}
		if !unicode.Is(xmlFirst, c) && !unicode.Is(xmlSecond, c) {
			return false
		}
	}
	return true
}

// xmlFirst and xmlSecond are the Unicode range tables for XML name
// characters, matching encoding/xml's internal "first" and "second" tables.
// xmlFirst covers NameStartChar, xmlSecond covers the additional NameChar.
var xmlFirst = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x003A, 0x003A, 1}, // :
		{0x0041, 0x005A, 1}, // A-Z
		{0x005F, 0x005F, 1}, // _
		{0x0061, 0x007A, 1}, // a-z
		{0x00C0, 0x00D6, 1},
		{0x00D8, 0x00F6, 1},
		{0x00F8, 0x02FF, 1},
		{0x0370, 0x037D, 1},
		{0x037F, 0x1FFF, 1},
		{0x200C, 0x200D, 1},
		{0x2070, 0x218F, 1},
		{0x2C00, 0x2FEF, 1},
		{0x3001, 0xD7FF, 1},
		{0xF900, 0xFDCF, 1},
		{0xFDF0, 0xFFFD, 1},
	},
	R32: []unicode.Range32{
		{0x10000, 0xEFFFF, 1},
	},
}

var xmlSecond = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x002D, 0x002E, 1}, // - .
		{0x0030, 0x0039, 1}, // 0-9
		{0x00B7, 0x00B7, 1},
		{0x0300, 0x036F, 1},
		{0x203F, 0x2040, 1},
	},
}

package shim

import (
	"bufio"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// nsStack tracks namespace prefix bindings for the encoder, matching
// stdlib encoding/xml's createAttrPrefix/popPrefix behavior.
type nsStack struct {
	attrPrefix map[string]string // URI → prefix
	attrNS     map[string]string // prefix → URI
	prefixes   []string          // stack; "" = scope marker
	seq        int               // collision suffix counter
}

func (s *nsStack) push() {
	s.prefixes = append(s.prefixes, "") // scope marker
}

func (s *nsStack) pop() {
	for len(s.prefixes) > 0 {
		p := s.prefixes[len(s.prefixes)-1]
		s.prefixes = s.prefixes[:len(s.prefixes)-1]
		if p == "" {
			break
		}
		if uri, ok := s.attrNS[p]; ok {
			delete(s.attrPrefix, uri)
			delete(s.attrNS, p)
		}
	}
}

// createAttrPrefix finds or creates a prefixed namespace binding for url.
// It writes the xmlns:prefix="url" declaration to w as a side effect.
func (s *nsStack) createAttrPrefix(w *bufio.Writer, url string) string {
	if s.attrPrefix != nil {
		if prefix := s.attrPrefix[url]; prefix != "" {
			return prefix
		}
	}

	if url == lexicon.NamespaceXML {
		return "xml"
	}

	if s.attrPrefix == nil {
		s.attrPrefix = make(map[string]string)
		s.attrNS = make(map[string]string)
	}

	// Derive candidate from URI: last path segment
	candidate := strings.TrimRight(url, "/")
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

	s.attrPrefix[url] = prefix
	s.attrNS[prefix] = url

	_, _ = w.WriteString(`xmlns:`)
	_, _ = w.WriteString(prefix)
	_, _ = w.WriteString(`="`)
	_ = escapeAttrVal(w, url)
	_, _ = w.WriteString(`" `)

	s.prefixes = append(s.prefixes, prefix)
	return prefix
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

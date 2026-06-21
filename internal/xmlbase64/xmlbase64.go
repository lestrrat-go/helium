// Package xmlbase64 decodes xs:base64Binary lexical values that may contain
// interspersed XML whitespace (space, tab, CR, LF). The xs:base64Binary lexical
// space permits such whitespace, and real-world XML Signature/Encryption
// producers routinely line-wrap and indent base64. Go's encoding/base64
// tolerates CR/LF but rejects space and tab, so all four are stripped first.
package xmlbase64

import (
	"encoding/base64"
	"strings"
)

// DecodeString strips the four XML whitespace characters from s and
// base64-decodes the result with StdEncoding. No other characters are
// removed, so invalid base64 still fails.
func DecodeString(s string) ([]byte, error) {
	if !strings.ContainsAny(s, " \t\r\n") {
		return base64.StdEncoding.DecodeString(s)
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := range len(s) {
		switch c := s[i]; c {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace
		default:
			b.WriteByte(c)
		}
	}
	return base64.StdEncoding.DecodeString(b.String())
}

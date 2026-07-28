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

// DecodedLen returns the number of bytes DecodeString produces for s,
// skipping the same XML whitespace DecodeString strips. It allocates
// nothing, so a caller can weigh a base64 value against a byte budget
// before paying for the decode.
//
// The count is exact for input DecodeString accepts. For input it rejects
// the result is meaningless — the caller learns that from the decode.
func DecodedLen(s string) int {
	var chars, pad int
	for i := range len(s) {
		switch c := s[i]; c {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace, as DecodeString does
		case '=':
			pad++
			chars++
		default:
			chars++
		}
	}
	n := chars/4*3 - pad
	if n < 0 {
		return 0
	}
	return n
}

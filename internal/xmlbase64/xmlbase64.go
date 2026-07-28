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

// DecodedLen counts the bytes a DecodeString of s would need, skipping the
// same XML whitespace DecodeString strips. It allocates nothing, so a caller
// can weigh a base64 value against a byte budget before paying for the decode.
//
// The count is exact for input DecodeString accepts.
//
// For input it rejects the count never falls below what the rejected decode
// costs: encoding/base64 sizes its output buffer from the character count
// alone and allocates it BEFORE validating a single character, so a count that
// trusted a lexical form the decoder will refuse would let an arbitrarily
// large value slip past a budget and still be allocated. Padding is therefore
// only deducted from a value whose whole lexical shape is otherwise decodable:
// a character count that is a multiple of four, at most two '=' and only at
// the end, and every other character inside the base64 alphabet. A value
// failing any of those is charged the full quantum count, which is exactly
// what the decoder allocates for it.
func DecodedLen(s string) int {
	var chars, pad int
	// decodableShape stays true only while s could still be a value the
	// decoder accepts, so the padding deduction below is safe to apply.
	decodableShape := true
	for i := range len(s) {
		switch c := s[i]; c {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace, as DecodeString does
		case '=':
			pad++
			chars++
		default:
			// Padding may only end a value, and every other character must be
			// in the alphabet, so either way this one cannot decode.
			if pad > 0 || !isAlphabet(c) {
				decodableShape = false
			}
			chars++
		}
	}
	// quanta is the buffer encoding/base64 allocates for chars characters.
	quanta := chars / 4 * 3
	if !decodableShape || pad > 2 || chars%4 != 0 {
		return quanta
	}
	// chars is 0 (so pad is 0) or at least 4, hence quanta >= 3 >= pad.
	return quanta - pad
}

// isAlphabet reports whether c is one of the 64 characters
// base64.StdEncoding decodes. Padding and XML whitespace are handled by the
// caller and are deliberately not alphabet characters.
func isAlphabet(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z':
		return true
	case c >= 'a' && c <= 'z':
		return true
	case c >= '0' && c <= '9':
		return true
	case c == '+', c == '/':
		return true
	}
	return false
}

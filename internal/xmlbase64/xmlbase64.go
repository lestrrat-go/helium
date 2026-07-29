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
//
// Its transient allocation tracks the base64 characters, never the lexical
// length: whitespace costs nothing beyond the scan. A caller weighing a value
// against a byte budget charges [Counter.DecodedLen], which counts those same
// characters and so never under-states what the decode allocates. Sizing the
// buffer on len(s) would let whitespace an attacker appends allocate memory no
// budget ever charged, growing without bound as the input grows.
func DecodeString(s string) ([]byte, error) {
	chars := charCount(s)
	if chars == len(s) {
		return base64.StdEncoding.DecodeString(s)
	}
	var b strings.Builder
	b.Grow(chars)
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

// AppendStripped appends to dst the bytes of src that DecodeString keeps,
// which is every byte that is not one of the four XML whitespace characters.
//
// A caller assembling a value that arrives in pieces builds only those
// characters, so what it holds tracks the base64 the decoder will see rather
// than the lexical length. Sizing dst with [Counter.Chars] — or, for a caller
// that stops as soon as its own limit is passed, with whatever ceiling that
// limit puts on the count — makes the assembly a single allocation.
func AppendStripped(dst, src []byte) []byte {
	for _, c := range src {
		switch c {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace, as DecodeString does
		default:
			dst = append(dst, c)
		}
	}
	return dst
}

// charCount counts the bytes of s that are not XML whitespace, which is
// exactly the run of characters DecodeString hands to the decoder. It
// allocates nothing.
func charCount(s string) int {
	var n int
	for i := range len(s) {
		switch s[i] {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace, as DecodeString does
		default:
			n++
		}
	}
	return n
}

// Counter counts what decoding a base64 value would need while the value
// arrives in pieces, so a caller holding it in fragments never has to
// concatenate them to weigh it against a byte budget. In XML a single
// xs:base64Binary value may be spread over any number of text and CDATA
// children, and the lexical text an attacker can attach to it is unrelated to
// the bytes it decodes to, so materializing the whole text first would
// allocate memory no budget has yet approved.
//
// The zero Counter is ready to use and allocates nothing. Feeding it the
// pieces in order gives exactly the counts of their concatenation.
//
// Counting each piece on its own and summing the results would not merely
// approximate that: it can report far less, because none of the padding,
// quantum, or alphabet rules distribute over a split. A thousand sibling "AAA"
// pieces each count 0 (three characters is not a whole quantum) against the
// 2250 of their concatenation. Carrying the running character count, the
// running padding count, and whether the shape can still decode across every
// piece is what makes the streaming count exact.
type Counter struct {
	chars int
	pad   int
	// undecodable records that the value can no longer be one the decoder
	// accepts, so the padding deduction in DecodedLen must not be applied.
	// It is stated negatively to keep the zero Counter usable.
	undecodable bool
}

// Add folds the next piece of the value into the counts. Pieces must be added
// in the order they appear in the value: padding is only padding at the end,
// and that is a property of the concatenation, not of any one piece.
//
// The piece is bytes because that is how a DOM hands out node content;
// converting it to a string first would copy the lexical length this counter
// exists to keep out of memory.
func (c *Counter) Add(piece []byte) {
	for _, ch := range piece {
		switch ch {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace, as DecodeString does
		case '=':
			c.pad++
			c.chars++
		default:
			// Padding may only end a value, and every other character must be
			// in the alphabet, so either way this one cannot decode.
			if c.pad > 0 || !isAlphabet(ch) {
				c.undecodable = true
			}
			c.chars++
		}
	}
}

// Chars reports the base64 characters added so far, which is exactly the run
// of bytes [AppendStripped] would produce for the same pieces and the run
// DecodeString hands the decoder.
func (c *Counter) Chars() int {
	return c.chars
}

// DecodedLen counts the bytes a DecodeString of the added pieces would need.
//
// The count is exact for a value DecodeString accepts.
//
// For one it rejects the count never falls below what the rejected decode
// costs: encoding/base64 sizes its output buffer from the character count
// alone and allocates it BEFORE validating a single character, so a count that
// trusted a lexical form the decoder will refuse would let an arbitrarily
// large value slip past a budget and still be allocated. Padding is therefore
// only deducted from a value whose whole lexical shape is otherwise decodable:
// a character count that is a multiple of four, at most two '=' and only at
// the end, and every other character inside the base64 alphabet. A value
// failing any of those is charged the full quantum count, which is exactly
// what the decoder allocates for it.
//
// This method is the only entry point to that count; there is deliberately no
// package-level DecodedLen(string). The characters of a value weighed this way
// — a CipherValue against a byte budget, an OAEPparams label against a fixed
// limit — arrive as separate child nodes and are never joined into one string
// before it is weighed, so no caller holds a whole value to pass. The
// single-value form is a zero-allocation two lines — var c Counter; c.Add(b) —
// and a package function would be an exported symbol in an internal package
// with no callers. Both claims are checkable: grep the tree for DecodedLen
// (every production call is in xmlenc1/parse.go), and measure c.Add with
// testing.AllocsPerRun.
func (c *Counter) DecodedLen() int {
	// quanta is the buffer encoding/base64 allocates for c.chars characters.
	quanta := c.chars / 4 * 3
	if c.undecodable || c.pad > 2 || c.chars%4 != 0 {
		return quanta
	}
	// chars is 0 (so pad is 0) or at least 4, hence quanta >= 3 >= pad.
	return quanta - c.pad
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

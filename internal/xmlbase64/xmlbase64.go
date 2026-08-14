// Package xmlbase64 decodes xs:base64Binary lexical values that may contain
// interspersed XML whitespace (space, tab, CR, LF). The xs:base64Binary lexical
// space permits such whitespace, and real-world XML Signature/Encryption
// producers routinely line-wrap and indent base64. Go's encoding/base64
// tolerates CR/LF but rejects space and tab, so all four are stripped first.
//
// A value in a document may also be spread over any number of text and CDATA
// children, so [DecodeElement] and [Counter] work straight off those children:
// a caller with a byte budget can weigh a value and charge for it before
// anything the size of its lexical text is ever built.
package xmlbase64

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

// DecodeElement charges charge for the base64 content of e's character-data
// children and then decodes it. It never builds the value's lexical text.
//
// That distinction is the whole point of taking an element in place of a
// string. xs:base64Binary permits XML whitespace between characters, and a
// value's text may arrive as any number of text and CDATA children, so the
// lexical length an attacker controls has no relation to the decoded bytes a
// budget charges. Joining the children into one string first would allocate
// that lexical length before the budget could refuse it, and would keep
// allocating it for every value the budget accepts, which leaves the accepted
// case unbounded as well.
//
// So one pass counts the value with a [Counter], which carries the counting
// state across each child boundary — a base64 quantum, and padding itself, may
// be split between children — and only then is the charge made. The second pass
// builds just the characters the decoder will see.
//
// Text, CDATA, and entity-reference children are read, and nothing else is.
// That is a bound, and not mere tidiness: an element child's Content is the
// concatenation of that child's WHOLE subtree, aggregated into one buffer, so
// reading a single element child would materialize an arbitrarily large tree
// before any charge — exactly what counting first exists to prevent.
//
// An element child is refused with an error and never skipped, because
// xs:base64Binary is a simple type and a simple content type admits only
// character information items. Neither ds:DigestValue (a restriction of
// xs:base64Binary) nor ds:SignatureValue (a simpleContent extension of it) may
// legally contain one, so refusing it is the more correct reading of the value
// as well as the bound. Every child kind not named here is refused with it.
//
// An entity reference IS read, because it stands for character data and a
// conforming document may write any part of a value as one. It is read through
// [entityReplacementText], which takes one bounded step to the declared
// replacement literal and no further, so admitting it reintroduces no
// aggregation. A replacement that is not base64 — a nested reference, which
// stays the unexpanded literal "&inner;", or markup — simply leaves the value
// undecodable, exactly as the same characters written inline would. A reference
// the parser linked no declaration to contributes nothing, which is what
// canonicalization makes of it too; see [entityReplacementText].
//
// Comment and processing-instruction children ARE skipped instead: per the XML
// infoset neither contributes a character information item, so splicing their
// text into the base64 stream would corrupt a legitimately commented value.
//
// What is held is therefore bounded by what charge approved: the stripped
// characters at most four thirds of it, the decoded bytes exactly it, and
// nothing at all scaling with the lexical length. The copy
// [helium.Text.Content] returns per child is the floor, so the peak is the
// largest single child, well under the whole value.
//
// charge's error is returned unchanged so a caller can pass its own
// resource-limit error straight through; every other error returned is a
// refused child or the encoding/base64 decode failure, for the caller to wrap
// with what it was decoding. A refused child is reported before the charge
// runs, and the charge precedes the decode, so a value that is both over budget
// and invalid base64 reports the charge error.
func DecodeElement(e *helium.Element, charge func(int) error) ([]byte, error) {
	var counter Counter
	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		content, err := characterData(child)
		if err != nil {
			return nil, err
		}
		counter.Add(content)
	}
	if err := charge(counter.DecodedLen()); err != nil {
		return nil, err
	}
	chars := make([]byte, 0, counter.Chars())
	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		// Every child was accepted by the counting pass above, so the only
		// error characterData can report here is one that pass already refused.
		content, err := characterData(child)
		if err != nil {
			return nil, err
		}
		chars = AppendStripped(chars, content)
	}
	// The same decode DecodeString performs, minus the copy converting the
	// already-stripped characters to a string, and sized at the count the
	// charge approved, in place of encoding/base64's own padding-blind
	// DecodedLen — see [Counter.DecodedLen].
	decoded := make([]byte, counter.DecodedLen())
	n, err := base64.StdEncoding.Decode(decoded, chars)
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}

// characterData returns the character data n contributes to an xs:base64Binary
// value: its content for a text or CDATA child, the referenced entity's
// declared replacement text for an entity reference that has one, nil for a
// child that contributes none — a comment, a processing instruction, or a
// reference to an entity nothing declared — and an error for a child a simple
// content type does not admit at all. [DecodeElement] documents why each kind
// lands where it does.
func characterData(n helium.Node) ([]byte, error) {
	if t, ok := helium.AsNode[*helium.Text](n); ok {
		return t.Content(), nil
	}
	if c, ok := helium.AsNode[*helium.CDATASection](n); ok {
		return c.Content(), nil
	}
	if r, ok := helium.AsNode[*helium.EntityRef](n); ok {
		return entityReplacementText(r)
	}
	if _, ok := helium.AsNode[*helium.Comment](n); ok {
		return nil, nil
	}
	if _, ok := helium.AsNode[*helium.ProcessingInstruction](n); ok {
		return nil, nil
	}
	return nil, fmt.Errorf("xs:base64Binary value has a non-character child (%s %q)", n.Type(), n.Name())
}

// entityReplacementText returns the declared replacement text of the entity r
// refers to, which is the character data r contributes to the value.
//
// Exactly one child is read — r's first, which the parser links to the declared
// [helium.Entity] — and that shape is what keeps the read bounded.
// [helium.Entity.Content] is a leaf accessor returning the raw replacement
// literal: it expands no nested reference and recurses into no subtree, so the
// cost is one copy of that literal, the same one-copy floor a text child
// already has. Reading r.Content() instead would aggregate r's children, which
// is bounded for a parser-built reference but not for a hand-built one carrying
// an element. Walking past the first child would leave the value altogether:
// the Entity's siblings are the remaining entity declarations of the DTD, not
// more of this value.
//
// A reference with NO child contributes no characters and is not an error. That
// is ordinary parser output, not a malformed tree: per XML 1.0 "Entity
// Declared", a general entity that nothing declares is a VALIDITY error rather
// than a well-formedness one once the document has an external subset the
// processor did not read, or a parameter-entity reference, and is not
// standalone="yes". A non-validating processor keeps going, so helium builds the
// reference with no declaration linked to it. [helium.EntityRef.Content] is
// likewise empty for it, and c14n canonicalizes such a reference by walking the
// children it does not have, so reading it as no characters is what agrees with
// the rest of the tree — a value whose base64 reader and canonicalizer disagreed
// about the same document would reject signatures the canonical form accepts.
//
// A NON-NIL first child that is not an [helium.Entity] is refused. The parser
// links a reference to its declared Entity or to nothing at all, so that shape
// is reachable only from a caller-assembled DOM, and refusing it is what holds
// the bound for one.
func entityReplacementText(r *helium.EntityRef) ([]byte, error) {
	first := r.FirstChild()
	if first == nil {
		return nil, nil
	}
	entity, ok := helium.AsNode[*helium.Entity](first)
	if !ok {
		return nil, fmt.Errorf("xs:base64Binary value has an entity reference whose first child is not an entity declaration (%s %q)", r.Type(), r.Name())
	}
	return entity.Content(), nil
}

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

// DecodedLen counts the decoded bytes of the added pieces.
//
// For a value the decoder accepts the count is exactly the decode's output
// length, so a buffer sized from it holds the whole decode with nothing to
// spare — which is what lets [DecodeElement] allocate precisely what it
// charged. It is NOT the buffer encoding/base64 itself would allocate for the
// same value: base64.DecodeString sizes from the character count alone and so
// over-allocates by the padding count, up to 2 bytes.
//
// For a value the decoder rejects the count never falls below what the rejected
// decode costs: encoding/base64 sizes its output buffer from the character
// count alone and allocates it BEFORE validating a single character, so a count
// that trusted a lexical form the decoder will refuse would let an arbitrarily
// large value slip past a budget and still be allocated. Padding is therefore
// only deducted from a value whose whole lexical shape is otherwise decodable:
// a character count that is a multiple of four, at most two '=' and only at
// the end, and every other character inside the base64 alphabet. A value
// failing any of those is charged the full quantum count, which is exactly
// what the decoder allocates for it.
//
// This method is the only entry point to that count; there is deliberately no
// package-level DecodedLen(string). The characters of a value weighed this way
// — a SignatureValue or DigestValue against a byte budget, a CipherValue
// against one, an OAEPparams label against a fixed limit — arrive as separate
// child nodes and are never joined into one string before it is weighed, so no
// caller holds a whole value to pass. The single-value form is a
// zero-allocation two lines — var c Counter; c.Add(b) — and a package function
// would be an exported symbol in an internal package with no callers. Both
// claims are checkable: grep the tree for DecodedLen (the charges in
// [DecodeElement] and xmlenc1/parse.go are the only production calls), and
// measure c.Add with testing.AllocsPerRun.
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

package helium

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// xmlNormalizationForm maps a normalization-form parameter name to its
// golang.org/x/text norm.Form and reports whether normalization is active. "NFC",
// "NFD", "NFKC", and "NFKD" enable it; "" and "none" disable it. Any other value
// also returns active=false here, but validNormalizationForm rejects it so WriteTo
// fails rather than silently disabling normalization (the caller — fn:serialize —
// rejects "fully-normalized" as SESU0011 before reaching the writer).
func xmlNormalizationForm(form string) (norm.Form, bool) {
	switch form {
	case "NFC":
		return norm.NFC, true
	case "NFD":
		return norm.NFD, true
	case "NFKC":
		return norm.NFKC, true
	case "NFKD":
		return norm.NFKD, true
	}
	return norm.NFC, false
}

// validNormalizationForm reports whether form is a normalization-form value the
// writer accepts: "" and "none" disable normalization; "NFC", "NFD", "NFKC", and
// "NFKD" enable it. Any other value (a typo, or a form the writer does not
// implement such as "fully-normalized") is rejected so WriteTo can surface
// ErrUnsupportedNormalizationForm instead of silently disabling normalization.
func validNormalizationForm(form string) bool {
	switch form {
	case "", "none", "NFC", "NFD", "NFKC", "NFKD":
		return true
	}
	return false
}

// contentSegment is one piece of a text or attribute node's character content
// after pre-normalization character-map matching: either a normalized run of
// non-mapped characters the escaper processes normally, or the replacement for
// one mapped input rune, which the caller emits verbatim.
type contentSegment struct {
	// text is the normalized non-mapped run (mapped is false).
	text []byte
	// repl is the character-map replacement for one mapped input rune (mapped
	// is true).
	repl   string
	mapped bool
}

// normalizeContent applies the writer's requested Unicode normalization to a text
// or attribute node's character content and returns it as segments.
// Character-map matching is decided on the PRE-normalization content —
// Serialization 3.1 §4 applies character mapping (rule c) before Unicode
// normalization (rule d) and never re-applies it — so a rune CREATED by
// normalization (e.g. NFC composing "e"+U+0301 into U+00E9) is ordinary content,
// never newly matched by the map. The content is split at each mapped input
// rune: every maximal run of non-mapped characters is normalized on its own and
// becomes a text segment, and each mapped rune becomes a replacement segment the
// caller emits verbatim (not re-escaped, not normalized), matching Serialization
// 3.1 §7. Splitting keeps replacements out of the escaped byte stream entirely,
// so matching cannot be perturbed by any content, and the escaper runs with no
// character map. It must only be called when d.normalize is true.
func (d *writeSession) normalizeContent(s []byte) []contentSegment {
	if len(d.charMap) == 0 {
		return []contentSegment{{text: d.normForm.Bytes(s)}}
	}
	var segs []contentSegment
	seg := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		repl, mapped := d.charMap[r]
		if !mapped {
			i += width
			continue
		}
		// Close the non-mapped run ending just before this mapped character,
		// then record the replacement for exactly this pre-normalization
		// occurrence.
		if i > seg {
			segs = append(segs, contentSegment{text: d.normForm.Bytes(s[seg:i])})
		}
		segs = append(segs, contentSegment{repl: repl, mapped: true})
		i += width
		seg = i
	}
	if segs == nil {
		// No mapped rune in the content: normalize it whole.
		return []contentSegment{{text: d.normForm.Bytes(s)}}
	}
	if seg < len(s) {
		segs = append(segs, contentSegment{text: d.normForm.Bytes(s[seg:])})
	}
	return segs
}

// writeReplacementSegment writes a character-map replacement verbatim, per
// Serialization 3.1 §7 (a replacement is never re-escaped or normalized). A
// non-ASCII replacement cannot be represented under a US-ASCII output encoding,
// so it is rejected when rejectNonASCII is set, mirroring
// writeCharMapReplacement on the non-normalizing path.
func writeReplacementSegment(w io.Writer, repl string, rejectNonASCII bool) error {
	if rejectNonASCII && hasNonASCII(repl) {
		return unsupportedASCIIErr("character-map replacement")
	}
	_, err := io.WriteString(w, repl)
	return err
}

// writeNormalizedText writes a text node's character content under an active
// Normalization request: each non-mapped segment is normalized and escaped as
// ordinary text with no character map (so a normalization-created rune is never
// newly matched), and each replacement segment is emitted verbatim. It must
// only be called when d.normalize is true.
func (d *writeSession) writeNormalizedText(out io.Writer, content []byte) error {
	for _, seg := range d.normalizeContent(content) {
		if seg.mapped {
			if err := writeReplacementSegment(out, seg.repl, d.asciiReject()); err != nil {
				return err
			}
			continue
		}
		if err := escapeText(out, seg.text, false, d.escapeNonASCII, d.asciiOutput, d.asciiReject(), !d.replaceInvalidChars, d.xml11, nil); err != nil {
			return err
		}
	}
	return nil
}

// writeAttrValueContent escapes an attribute value's character content, applying
// the requested Unicode normalization (scoped to attribute nodes) and character
// maps. Shared by the generic and XHTML serialization paths. Under an active
// Normalization request, character-map matches are decided on the
// pre-normalization content: normalizeContent splits the content at each mapped
// rune, so a normalization-created rune is never newly matched and each
// replacement is emitted verbatim.
func (d *writeSession) writeAttrValueContent(out io.Writer, content []byte) error {
	if !d.normalize {
		return escapeAttrValue(out, content, d.escapeNonASCII, d.asciiOutput, d.asciiReject(), !d.replaceInvalidChars, d.xml11, d.charMap)
	}
	for _, seg := range d.normalizeContent(content) {
		if seg.mapped {
			if err := writeReplacementSegment(out, seg.repl, d.asciiReject()); err != nil {
				return err
			}
			continue
		}
		if err := escapeAttrValue(out, seg.text, d.escapeNonASCII, d.asciiOutput, d.asciiReject(), !d.replaceInvalidChars, d.xml11, nil); err != nil {
			return err
		}
	}
	return nil
}

var (
	qch_dquote = []byte{'"'}
	qch_quote  = []byte{'\''}
)

// ErrInvalidXMLChar is returned by the writer when a character in the tree is
// not valid in the target XML version and the writer is configured to reject
// (rather than replace) such characters via Writer.RejectInvalidChars. It maps
// to the XSLT/XQuery serialization error SERE0006.
var ErrInvalidXMLChar = errors.New("character is not valid in the target XML version")

func dumpQuotedString(out io.Writer, s string) error {
	dqi := strings.IndexByte(s, qch_dquote[0])
	if dqi < 0 {
		if _, err := out.Write(qch_dquote); err != nil {
			return err
		}
		if _, err := io.WriteString(out, s); err != nil {
			return err
		}
		if _, err := out.Write(qch_dquote); err != nil {
			return err
		}
		return nil
	}

	if qi := strings.IndexByte(s, qch_quote[0]); qi < 0 {
		if _, err := out.Write(qch_quote); err != nil {
			return err
		}
		if _, err := io.WriteString(out, s); err != nil {
			return err
		}
		if _, err := out.Write(qch_quote); err != nil {
			return err
		}
		return nil
	}

	if _, err := out.Write(qch_dquote); err != nil {
		return err
	}
	for len(s) > 0 && dqi > -1 {
		if _, err := io.WriteString(out, s[:dqi]); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "&quot;"); err != nil {
			return err
		}
		s = s[dqi+1:]
		dqi = strings.IndexByte(s, qch_dquote[0])
	}

	if len(s) > 0 {
		if _, err := io.WriteString(out, s); err != nil {
			return err
		}
	}
	if _, err := out.Write(qch_dquote); err != nil {
		return err
	}
	return nil
}

var (
	esc_quot = []byte("&quot;")
	esc_amp  = []byte("&amp;")
	esc_lt   = []byte("&lt;")
	esc_gt   = []byte("&gt;")
	esc_tab  = []byte("&#9;")
	esc_nl   = []byte("&#10;")
	esc_cr   = []byte("&#13;")

	esc_fffd     = []byte("\uFFFD")
	esc_fffd_ref = []byte("&#xFFFD;")
)

const upperHex = "0123456789ABCDEF"

// isXML11RestrictedChar reports whether r is an XML 1.1 restricted character: a
// control character that is a valid XML 1.1 Char but must be serialized as a
// character reference rather than appearing literally (XML 1.1 §2.11). Tab
// (U+0009), LF (U+000A), and CR (U+000D) are excluded — they follow the ordinary
// escaping rules.
func isXML11RestrictedChar(r rune) bool {
	switch {
	case r >= 0x1 && r <= 0x8:
		return true
	case r == 0xB || r == 0xC:
		return true
	case r >= 0xE && r <= 0x1F:
		return true
	case r >= 0x7F && r <= 0x84:
		return true
	case r >= 0x86 && r <= 0x9F:
		return true
	default:
		return false
	}
}

// isXML11SerializeAsCharRef reports whether r must be written as a character
// reference (rather than literally) when producing XML 1.1 output. This is the
// XML 1.1 RestrictedChar set (isXML11RestrictedChar) PLUS the two end-of-line
// characters NEL (U+0085) and LINE SEPARATOR (U+2028). Both are excluded from
// RestrictedChar, but XML 1.1 §2.11 line-ending normalization translates them to
// U+000A on input, so a literal occurrence would not round-trip; emitting them as
// character references preserves the value. In XML 1.0 neither is a line-ending
// character, so this is gated on the xml11 flag and 1.0 serialization is
// unaffected.
func isXML11SerializeAsCharRef(r rune) bool {
	return isXML11RestrictedChar(r) || r == 0x85 || r == 0x2028
}

// decimalCharRef writes r as a decimal character reference ("&#N;") into buf and
// returns the populated slice.
func decimalCharRef(buf *[12]byte, r rune) []byte {
	n := len(buf)
	n--
	buf[n] = ';'
	v := int(r)
	if v <= 0 {
		n--
		buf[n] = '0'
	}
	for v > 0 {
		n--
		buf[n] = byte('0' + v%10)
		v /= 10
	}
	n--
	buf[n] = '#'
	n--
	buf[n] = '&'
	return buf[n:]
}

func hexCharRef(buf *[8]byte, r rune) []byte {
	buf[0] = '&'
	buf[1] = '#'
	buf[2] = 'x'
	n := 3
	if r >= 0x10 {
		buf[n] = upperHex[(r>>4)&0x0F]
		n++
	}
	buf[n] = upperHex[r&0x0F]
	n++
	buf[n] = ';'
	n++
	return buf[:n]
}

// hexCharRefWide writes r as an uppercase-hex character reference ("&#xNN...;")
// into buf, handling the full XML character range (up to U+10FFFF, six hex
// digits). hexCharRef is limited to two digits (r <= 0xFF); this variant is used
// for US-ASCII output, where every non-ASCII character — including astral and BMP
// characters beyond Latin-1 — must be emitted as a reference.
func hexCharRefWide(buf *[10]byte, r rune) []byte {
	n := len(buf)
	n--
	buf[n] = ';'
	v := int(r)
	if v == 0 {
		n--
		buf[n] = '0'
	}
	for v > 0 {
		n--
		buf[n] = upperHex[v&0x0F]
		v >>= 4
	}
	n--
	buf[n] = 'x'
	n--
	buf[n] = '#'
	n--
	buf[n] = '&'
	return buf[n:]
}

func isInCharacterRange(r rune) bool {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

// isSerializableChar reports whether r is a character the writer may serialize
// for the target XML version: any XML 1.0 Char (isInCharacterRange) plus, when
// targeting XML 1.1, the restricted control characters (isXML11SerializeAsCharRef)
// that are valid in 1.1 but must be emitted as character references. A character
// failing this is rejected with ErrInvalidXMLChar when RejectInvalidChars is set.
func isSerializableChar(r rune, xml11 bool) bool {
	return isInCharacterRange(r) || (xml11 && isXML11SerializeAsCharRef(r))
}

// serializeRefFree screens content bound for a REFERENCE-LESS serialization
// context — comment text, processing-instruction data, and CDATA-section content.
// None of these admits a character reference, so a character must be serializable
// LITERALLY. For XML 1.0 output the literal set is the XML 1.0 Char range
// (isInCharacterRange). For XML 1.1 output it is that range MINUS the
// RestrictedChar set (isXML11RestrictedChar): element/attribute text may carry a
// RestrictedChar only as a character reference (XML 1.1 production [1] forbids a
// literal occurrence anywhere in the document), and these contexts have no
// reference form, so a RestrictedChar has no valid form here at all. The C1 half
// of that set (0x7F-0x84, 0x86-0x9F) is a legal literal XML 1.0 Char, so 1.0
// output keeps accepting it unchanged. The 1.1 screen is deliberately
// RestrictedChar, NOT isXML11SerializeAsCharRef: NEL (U+0085) and LINE SEPARATOR
// (U+2028) are not RestrictedChar and are legal literally in a 1.1 comment/PI/
// CDATA section (they are merely line-end-normalized to LF on re-parse), so
// rejecting them would refuse a well-formed document.
//
// A character failing the screen is a serialization error: under the default
// policy it is rejected with a sticky ErrInvalidXMLChar (returning stop=true, with
// nothing to write); under RejectInvalidChars(false) it is replaced by U+FFFD. A
// byte that is not valid UTF-8 is always replaced by U+FFFD, matching the
// text/attribute escapers (it has no character-reference form either). When the
// content is already clean it returns b unchanged so no copy is made.
func (s *writeSession) serializeRefFree(what string, b []byte) (out []byte, stop bool) {
	work := false
	for i := 0; i < len(b); {
		r, width := utf8.DecodeRune(b[i:])
		switch {
		case r == utf8.RuneError && width == 1:
			// A malformed UTF-8 byte is always replaced with U+FFFD.
			work = true
		case !isInCharacterRange(r) || (s.xml11 && isXML11RestrictedChar(r)):
			if !s.replaceInvalidChars {
				s.check(fmt.Errorf("helium: %s contains a character invalid in the target XML version: %w", what, ErrInvalidXMLChar))
				return nil, true
			}
			work = true
		}
		i += width
	}
	if !work {
		return b, false
	}
	out = make([]byte, 0, len(b))
	for i := 0; i < len(b); {
		r, width := utf8.DecodeRune(b[i:])
		if (r == utf8.RuneError && width == 1) || !isInCharacterRange(r) || (s.xml11 && isXML11RestrictedChar(r)) {
			out = append(out, esc_fffd...)
		} else {
			out = append(out, b[i:i+width]...)
		}
		i += width
	}
	return out, false
}

// dtdLiteral applies the reference-less serialization policy (serializeRefFree) to
// a DTD literal supplied as a string: an entity value, an external-ID system or
// public literal, or an enumeration token. A DTD literal admits no character
// reference for its LITERAL runes, so serializeRefFree's version-aware
// reference-less screen applies to every DTD literal site — a literal XML 1.1
// RestrictedChar (an EntityValue could in principle carry it as a character
// reference instead, but the writer does not rewrite literals into references) is
// rejected (default) or U+FFFD-substituted (RejectInvalidChars(false)). It
// returns the value to emit and whether the caller should stop because the default
// reject policy recorded a sticky ErrInvalidXMLChar.
func (s *writeSession) dtdLiteral(what, value string) (string, bool) {
	if value == "" {
		return value, false
	}
	out, stop := s.serializeRefFree(what, []byte(value))
	if stop {
		return "", true
	}
	return string(out), false
}

// entityValueLiteral applies the DTD-literal policy to an entity-VALUE literal and
// additionally validates every character reference the value carries. Unlike an
// external-ID or enumeration literal (plain dtdLiteral), an EntityValue is subject
// to reference recognition (XML §4.4.5): a &#N; / &#xN; is a real character
// reference whose target must be serializable in the target XML version. So the
// literal runes are screened by dtdLiteral (reference-less rule) and every numeric
// character reference is validated by screenCharRefs against isSerializableChar —
// a 1.1 EntityValue may reference a RestrictedChar (legal as &#1; under a 1.1
// target), a 1.0 one may not. An invalid target is rejected (default) or its
// reference replaced by the U+FFFD representation (RejectInvalidChars(false)).
func (s *writeSession) entityValueLiteral(what, value string) (string, bool) {
	lit, stop := s.dtdLiteral(what, value)
	if stop {
		return "", true
	}
	if strings.IndexByte(lit, '&') == -1 {
		return lit, false
	}
	return s.screenCharRefs(what, lit)
}

// screenCharRefs validates every numeric character reference ("&#N;" / "&#xN;") in
// an entity value, leaving named references (&amp;, &e;) and non-reference text
// untouched. A reference whose target is serializable in the target XML version
// (isSerializableChar) is preserved verbatim. An invalid target is a serialization
// error: under the default policy it records a sticky ErrInvalidXMLChar (returning
// stop=true); under RejectInvalidChars(false) the whole reference is replaced by
// the U+FFFD representation — the &#xFFFD; reference when non-ASCII characters are
// being escaped (EscapeNonASCII / US-ASCII output), the raw U+FFFD character
// otherwise (matching the text/attribute escapers). A "&#…" sequence whose body is
// syntactically numeric but out of range (> U+10FFFF, e.g. "&#x110000;") is a
// CharRef with an invalid CHARACTER target and is handled like a non-serializable
// one — rejected or replaced. A non-numeric "&#…" sequence (a name-grammar matter
// outside the character policy) is left verbatim. When nothing is replaced it
// returns value unchanged so no copy is retained.
func (s *writeSession) screenCharRefs(what, value string) (string, bool) {
	var b strings.Builder
	last := 0
	i := 0
	for i < len(value) {
		if value[i] != '&' || i+1 >= len(value) || value[i+1] != '#' {
			i++
			continue
		}
		semi := strings.IndexByte(value[i:], ';')
		if semi == -1 {
			break
		}
		refEnd := i + semi + 1
		cp, kind := parseCharRefBody(value[i+2 : i+semi])
		if kind == charRefNotNumeric {
			i += 2
			continue
		}
		if kind == charRefInRange && isSerializableChar(cp, s.xml11) {
			i = refEnd
			continue
		}
		if !s.replaceInvalidChars {
			s.check(fmt.Errorf("helium: %s contains a character reference to a character invalid in the target XML version: %w", what, ErrInvalidXMLChar))
			return "", true
		}
		b.WriteString(value[last:i])
		if s.escapeNonASCII || s.asciiOutput {
			b.Write(esc_fffd_ref)
		} else {
			b.Write(esc_fffd)
		}
		last = refEnd
		i = refEnd
	}
	if last == 0 {
		return value, false
	}
	b.WriteString(value[last:])
	return b.String(), false
}

// writeCharMapReplacement flushes s[last:cut] to w and writes the raw
// (unescaped) character-map replacement, returning the new value of last (the
// byte offset just past the mapped character). It is shared by escapeText and
// escapeAttrValue so a character map substitutes a mapped rune with its literal
// replacement string, per XSLT/XQuery Serialization 3.1 §7 (character maps are
// applied as the final step and the replacement is emitted verbatim, not
// re-escaped).
//
// A character-map replacement is never re-escaped, so a non-ASCII replacement
// would leak raw UTF-8 under a US-ASCII output encoding. When rejectNonASCII is
// set (the octet-producing US-ASCII path, not declaration-only fn:serialize)
// such a replacement is rejected with a labelled early error before anything is
// written; the output-writer net is the backstop.
func writeCharMapReplacement(w io.Writer, s []byte, last, cut, next int, repl string, rejectNonASCII bool) (int, error) {
	if rejectNonASCII && hasNonASCII(repl) {
		return last, unsupportedASCIIErr("character-map replacement")
	}
	if _, err := w.Write(s[last:cut]); err != nil {
		return last, err
	}
	if _, err := io.WriteString(w, repl); err != nil {
		return last, err
	}
	return next, nil
}

func escapeAttrValue(w io.Writer, s []byte, escapeNonASCII, asciiOutput, rejectCharMapNonASCII, rejectInvalidChars, xml11 bool, charMap map[rune]string) error {
	var esc []byte
	var hbuf [8]byte
	var wbuf [10]byte
	var dbuf [12]byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		// The character-map lookup deliberately PRECEDES the invalid-char rejection
		// below: Serialization 3.1 §5.1.11 applies character maps before character
		// checking, and SERE0006 is defined on the SERIALIZED result — after mapping,
		// a mapped invalid character is gone, so no forbidden character survives to
		// reject. Mapping an invalid char to a safe replacement is explicit per-rune
		// configuration (fn:serialize use-character-maps), not silent mutation.
		if repl, ok := charMap[r]; ok {
			newLast, err := writeCharMapReplacement(w, s, last, i-width, i, repl, rejectCharMapNonASCII)
			if err != nil {
				return err
			}
			last = newLast
			continue
		}
		switch r {
		case '"':
			esc = esc_quot
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		case '\n':
			esc = esc_nl
		case '\r':
			esc = esc_cr
		case '\t':
			esc = esc_tab
		default:
			// A character outside the XML character range (e.g. a C0/C1 control
			// char) is a serialization error when rejection is enabled — checked
			// before every emission branch so it is caught regardless of the
			// escaping setting. A valid XML 1.1 restricted character is exempt: it
			// is in-range for the target version and serializes as a character
			// reference below. A malformed UTF-8 byte decodes to U+FFFD, which is
			// IN range, so it is not a version error here.
			if rejectInvalidChars && !isSerializableChar(r, xml11) {
				return ErrInvalidXMLChar
			}
			// XML 1.1 restricted control characters (and the NEL/LINE SEPARATOR
			// end-of-line characters) are valid but may not appear literally: emit
			// them as decimal character references (before the out-of-range
			// replacement and the escapeNonASCII hex branch).
			if xml11 && isXML11SerializeAsCharRef(r) {
				esc = decimalCharRef(&dbuf, r)
				break
			}
			// A character outside the XML character range (or a lone U+FFFD from a
			// malformed UTF-8 byte) has no valid literal or numeric-reference form,
			// so in replacement mode it becomes U+FFFD — the &#xFFFD; reference
			// when non-ASCII characters are being escaped (matching libxml2) and
			// the raw replacement character otherwise. Checked BEFORE the
			// escapeNonASCII / asciiOutput char-reference branches so an
			// out-of-range char never serializes as a bogus reference (e.g. &#x1;).
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				if escapeNonASCII || asciiOutput {
					esc = esc_fffd_ref
				} else {
					esc = esc_fffd
				}
				break
			}
			// US-ASCII output cannot represent a non-ASCII character literally, so
			// every valid non-ASCII character is emitted as a hex character reference
			// (the full range, not just Latin-1) — the octets stay pure US-ASCII,
			// consistent with the encoding declaration. Checked before the Latin-1-only
			// escapeNonASCII branch so BMP/astral characters are covered too.
			if asciiOutput && r >= 0x80 {
				esc = hexCharRefWide(&wbuf, r)
				break
			}
			if escapeNonASCII && !(0x20 <= r && r < 0x80) { //nolint:staticcheck
				if r < 0x100 {
					esc = hexCharRef(&hbuf, r)
					break
				}
			}
			// A genuine in-range U+FFFD is emitted as a character reference.
			if r == 0xFFFD {
				esc = esc_fffd_ref
				break
			}
			continue
		}

		if _, err := w.Write(s[last : i-width]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i
	}

	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

func escapeText(w io.Writer, s []byte, escapeNewline, escapeNonASCII, asciiOutput, rejectCharMapNonASCII, rejectInvalidChars, xml11 bool, charMap map[rune]string) error {
	var esc []byte
	var hbuf [8]byte
	var wbuf [10]byte
	var dbuf [12]byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		// The character-map lookup deliberately PRECEDES the invalid-char rejection
		// below: Serialization 3.1 §5.1.11 applies character maps before character
		// checking, and SERE0006 is defined on the SERIALIZED result — after mapping,
		// a mapped invalid character is gone, so no forbidden character survives to
		// reject. Mapping an invalid char to a safe replacement is explicit per-rune
		// configuration (fn:serialize use-character-maps), not silent mutation.
		if repl, ok := charMap[r]; ok {
			newLast, err := writeCharMapReplacement(w, s, last, i-width, i, repl, rejectCharMapNonASCII)
			if err != nil {
				return err
			}
			last = newLast
			continue
		}
		switch r {
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		case '\n':
			if !escapeNewline {
				continue
			}
			esc = esc_nl
		case '\r':
			esc = esc_cr
		default:
			// A character outside the XML character range (e.g. a C0/C1 control
			// char) is a serialization error when rejection is enabled — checked
			// before every emission branch so it is caught regardless of the
			// escaping setting. A valid XML 1.1 restricted character is exempt: it
			// is in-range for the target version and serializes as a character
			// reference below. A malformed UTF-8 byte decodes to U+FFFD, which is
			// IN range, so it is not a version error here.
			if rejectInvalidChars && !isSerializableChar(r, xml11) {
				return ErrInvalidXMLChar
			}
			// XML 1.1 restricted control characters (and the NEL/LINE SEPARATOR
			// end-of-line characters) are valid but may not appear literally: emit
			// them as decimal character references (before the out-of-range
			// replacement and the escapeNonASCII hex branch).
			if xml11 && isXML11SerializeAsCharRef(r) {
				esc = decimalCharRef(&dbuf, r)
				break
			}
			// A character outside the XML character range (or a lone U+FFFD from a
			// malformed UTF-8 byte) has no valid literal or numeric-reference form,
			// so in replacement mode it becomes U+FFFD — the &#xFFFD; reference
			// when non-ASCII characters are being escaped (matching libxml2) and
			// the raw replacement character otherwise. Checked BEFORE the
			// escapeNonASCII / asciiOutput char-reference branches so an
			// out-of-range char never serializes as a bogus reference (e.g. &#x1;).
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				if escapeNonASCII || asciiOutput {
					esc = esc_fffd_ref
				} else {
					esc = esc_fffd
				}
				break
			}
			// US-ASCII output cannot represent a non-ASCII character literally, so
			// every valid non-ASCII character is emitted as a hex character reference
			// (the full range, not just Latin-1) — the octets stay pure US-ASCII,
			// consistent with the encoding declaration. Checked before the Latin-1-only
			// escapeNonASCII branch so BMP/astral characters are covered too.
			if asciiOutput && r >= 0x80 {
				esc = hexCharRefWide(&wbuf, r)
				break
			}
			if escapeNonASCII && !(r == '\t' || (0x20 <= r && r < 0x80)) { //nolint:staticcheck
				if r < 0x100 {
					esc = hexCharRef(&hbuf, r)
					break
				}
			}
			// A genuine in-range U+FFFD is emitted as a character reference.
			if r == 0xFFFD {
				esc = esc_fffd_ref
				break
			}
			continue
		}

		if _, err := w.Write(s[last : i-width]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i
	}

	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

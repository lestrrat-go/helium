package helium

import (
	"bytes"
	"errors"
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

// normalizeContent applies the writer's requested Unicode normalization to a text
// or attribute node's character content while leaving character-map replacement
// spans inert. When a character map is in force, each maximal run of non-mapped
// characters is normalized on its own and every mapped character is copied
// through UNCHANGED, so the caller still passes the character map to the escaper
// — which substitutes the mapped character with its replacement verbatim (not
// re-escaped, not normalized), matching Serialization 3.1 §7. Normalizing around
// (rather than through) a mapped character keeps the replacement byte-identical
// regardless of the requested form. It must only be called when d.normalize is
// true.
func (d *writeSession) normalizeContent(s []byte) []byte {
	if len(d.charMap) == 0 {
		return d.normForm.Bytes(s)
	}
	var b bytes.Buffer
	b.Grow(len(s))
	seg := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		if _, ok := d.charMap[r]; !ok {
			i += width
			continue
		}
		// Normalize the non-mapped run ending just before this mapped character,
		// then copy the mapped character through unchanged so the escaper still
		// recognizes it and emits its replacement verbatim.
		if i > seg {
			b.Write(d.normForm.Bytes(s[seg:i]))
		}
		b.Write(s[i : i+width])
		i += width
		seg = i
	}
	if seg < len(s) {
		b.Write(d.normForm.Bytes(s[seg:]))
	}
	return b.Bytes()
}

// writeAttrValueContent escapes an attribute value's character content, applying
// the requested Unicode normalization (scoped to attribute nodes) and character
// maps. Shared by the generic and XHTML serialization paths.
func (d *writeSession) writeAttrValueContent(out io.Writer, content []byte) error {
	if d.normalize {
		// normalizeContent normalizes only the non-mapped runs and leaves mapped
		// characters in place, so the character map still drives the escaper to
		// emit each replacement verbatim.
		content = d.normalizeContent(content)
	}
	return escapeAttrValue(out, content, d.escapeNonASCII, d.asciiOutput, d.asciiReject(), d.rejectInvalidChars, d.xml11, d.charMap)
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
		r >= 0x20 && r <= 0xDF77 ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
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
			// before the escapeNonASCII char-reference branch so it is caught
			// regardless of that setting. A malformed UTF-8 byte decodes to
			// U+FFFD, which is IN range, so it is not a version error here.
			if rejectInvalidChars && !isInCharacterRange(r) {
				return ErrInvalidXMLChar
			}
			// XML 1.1 restricted control characters (and the NEL/LINE SEPARATOR
			// end-of-line characters) are valid but may not appear literally: emit
			// them as decimal character references (before the escapeNonASCII hex
			// branch and the out-of-range replacement).
			if xml11 && isXML11SerializeAsCharRef(r) {
				esc = decimalCharRef(&dbuf, r)
				break
			}
			// US-ASCII output cannot represent a non-ASCII character literally, so
			// every valid non-ASCII character is emitted as a hex character reference
			// (the full range, not just Latin-1) — the octets stay pure US-ASCII,
			// consistent with the encoding declaration. Checked before the Latin-1-only
			// escapeNonASCII branch so BMP/astral characters are covered too.
			if asciiOutput && r >= 0x80 && isInCharacterRange(r) {
				esc = hexCharRefWide(&wbuf, r)
				break
			}
			if escapeNonASCII && !(0x20 <= r && r < 0x80) { //nolint:staticcheck
				if r < 0x100 {
					esc = hexCharRef(&hbuf, r)
					break
				}
			}
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
				break
			}
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
			// before the escapeNonASCII char-reference branch so it is caught
			// regardless of that setting. A malformed UTF-8 byte decodes to
			// U+FFFD, which is IN range, so it is not a version error here.
			if rejectInvalidChars && !isInCharacterRange(r) {
				return ErrInvalidXMLChar
			}
			// XML 1.1 restricted control characters (and the NEL/LINE SEPARATOR
			// end-of-line characters) are valid but may not appear literally: emit
			// them as decimal character references (before the escapeNonASCII hex
			// branch and the out-of-range replacement).
			if xml11 && isXML11SerializeAsCharRef(r) {
				esc = decimalCharRef(&dbuf, r)
				break
			}
			// US-ASCII output cannot represent a non-ASCII character literally, so
			// every valid non-ASCII character is emitted as a hex character reference
			// (the full range, not just Latin-1) — the octets stay pure US-ASCII,
			// consistent with the encoding declaration. Checked before the Latin-1-only
			// escapeNonASCII branch so BMP/astral characters are covered too.
			if asciiOutput && r >= 0x80 && isInCharacterRange(r) {
				esc = hexCharRefWide(&wbuf, r)
				break
			}
			if escapeNonASCII && !(r == '\t' || (0x20 <= r && r < 0x80)) { //nolint:staticcheck
				if r < 0x100 {
					esc = hexCharRef(&hbuf, r)
					break
				}
			}
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
				break
			}
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

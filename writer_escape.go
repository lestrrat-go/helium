package helium

import (
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

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

func isInCharacterRange(r rune) bool {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xDF77 ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

func escapeAttrValue(w io.Writer, s []byte, escapeNonASCII, rejectInvalidChars, xml11 bool) error {
	var esc []byte
	var hbuf [8]byte
	var dbuf [12]byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
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
			// XML 1.1 restricted control characters are valid but may not appear
			// literally: emit them as decimal character references (before the
			// escapeNonASCII hex branch and the out-of-range replacement).
			if xml11 && isXML11RestrictedChar(r) {
				esc = decimalCharRef(&dbuf, r)
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

func escapeText(w io.Writer, s []byte, escapeNewline, escapeNonASCII, rejectInvalidChars, xml11 bool) error {
	var esc []byte
	var hbuf [8]byte
	var dbuf [12]byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
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
			// XML 1.1 restricted control characters are valid but may not appear
			// literally: emit them as decimal character references (before the
			// escapeNonASCII hex branch and the out-of-range replacement).
			if xml11 && isXML11RestrictedChar(r) {
				esc = decimalCharRef(&dbuf, r)
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

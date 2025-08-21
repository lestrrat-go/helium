package s11n

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

var (
	qch_dquote = []byte{'"'}
	qch_quote  = []byte{'\''}
)

// isInCharacterRange checks if rune is in XML Character Range
func isInCharacterRange(r rune) bool {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xDF77 ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

func DumpQuotedString(out io.Writer, s string) error {
	dqi := strings.IndexByte(s, qch_dquote[0])
	if dqi < 0 {
		// double quote is allowed, cool!
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
		// single quotes, then
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

	// Grr, can't use " or '. Well, let's escape all the double
	// quotes to &quot;, and quote the string

	if _, err := out.Write(qch_dquote); err != nil {
		return err
	}
	for len(s) > 0 && dqi > -1 {
		if _, err := io.WriteString(out, s[:dqi]); err != nil {
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
	esc_quot = []byte("&#34;") // shorter than "&quot;"
	// esc_apos = []byte("&#39;") // shorter than "&apos;"
	esc_amp  = []byte("&amp;")
	esc_lt   = []byte("&lt;")
	esc_gt   = []byte("&gt;")
	esc_tab  = []byte("&#9;")
	esc_nl   = []byte("&#10;")
	esc_cr   = []byte("&#13;")
	esc_fffd = []byte("\uFFFD") // Unicode replacement character
)

func EscapeAttrValue(w io.Writer, s []byte) error {
	// Note: Logging disabled for s11n functions as they lack context
	// TODO: Add context-aware versions if needed for debugging
	var esc []byte
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
			if !(0x20 <= r && r < 0x80) { // nolint:staticcheck
				if r < 0xE0 {
					esc = []byte(fmt.Sprintf("&#x%X;", r))
					break
				}
			}
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
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

// EscapeText writes to w the properly escaped XML equivalent
// of the plain text data s. If escapeNewline is true, newline
// characters will be escaped.
func EscapeText(w io.Writer, s []byte, escapeNewline bool) error {
	// Note: Logging disabled for s11n functions as they lack context
	// TODO: Add context-aware versions if needed for debugging
	var esc []byte
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
			if !(r == '\t' || (0x20 <= r && r < 0x80)) { // nolint:staticcheck
				if r < 0xE0 {
					esc = []byte(fmt.Sprintf("&#x%X;", r))
					break
				}
			}
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = esc_fffd
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

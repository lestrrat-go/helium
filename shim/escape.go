package shim

import (
	"io"
	"unicode/utf8"
)

var (
	escQuot = []byte("&#34;")
	escApos = []byte("&#39;")
	escAmp  = []byte("&amp;")
	escLT   = []byte("&lt;")
	escGT   = []byte("&gt;")
	escTab  = []byte("&#x9;")
	escNL   = []byte("&#xA;")
	escCR   = []byte("&#xD;")
	escFFFD = []byte("\uFFFD")
)

func isInCharacterRange(r rune) bool {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

// escapeText writes b to w with XML escaping for text content.
func escapeText(w io.Writer, b []byte) error {
	return doEscape(w, b, true)
}

// escapeAttrVal writes s to w with XML escaping for attribute values.
// Newlines ARE escaped (matching stdlib encoder behavior for attributes).
func escapeAttrVal(w io.Writer, s string) error {
	return doEscape(w, []byte(s), true)
}

// escapeString writes s to w with XML escaping for text content.
func escapeString(w io.Writer, s string) error {
	return doEscape(w, []byte(s), false)
}

func doEscape(w io.Writer, s []byte, escapeNewline bool) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch r {
		case '"':
			esc = escQuot
		case '\'':
			esc = escApos
		case '&':
			esc = escAmp
		case '<':
			esc = escLT
		case '>':
			esc = escGT
		case '\t':
			esc = escTab
		case '\n':
			if !escapeNewline {
				continue
			}
			esc = escNL
		case '\r':
			esc = escCR
		default:
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = escFFFD
			} else {
				continue
			}
		}
		if _, err := w.Write(s[last : i-width]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i
	}
	_, err := w.Write(s[last:])
	return err
}

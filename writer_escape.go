package helium

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/pdebug"
)

var (
	qch_dquote = []byte{'"'}
	qch_quote  = []byte{'\''}
)

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

func escapeAttrValue(w io.Writer, s []byte, escapeNonASCII bool) error {
	if pdebug.Enabled {
		debugbuf := bytes.Buffer{}
		w = io.MultiWriter(w, &debugbuf)
		g := pdebug.Marker("escapeAttrValue '%s'", s)
		defer func() {
			pdebug.Printf("escaped value '%s'", debugbuf.Bytes())
			g.End()
		}()
	}
	var esc []byte
	var hbuf [8]byte
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
			if escapeNonASCII && !(0x20 <= r && r < 0x80) { // nolint:staticcheck
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

func escapeText(w io.Writer, s []byte, escapeNewline bool, escapeNonASCII bool) error {
	if pdebug.Enabled {
		debugbuf := bytes.Buffer{}
		w = io.MultiWriter(w, &debugbuf)
		g := pdebug.IPrintf("START escapeText = '%s'", s)
		defer func() {
			g.IRelease("END escapeText = '%s'", debugbuf.Bytes())
		}()
	}
	var esc []byte
	var hbuf [8]byte
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
			if escapeNonASCII && !(r == '\t' || (0x20 <= r && r < 0x80)) { // nolint:staticcheck
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

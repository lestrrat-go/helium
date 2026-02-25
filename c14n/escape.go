package c14n

import (
	"io"
	"unicode/utf8"
)

var (
	escAmp  = []byte("&amp;")
	escLT   = []byte("&lt;")
	escGT   = []byte("&gt;")
	escQuot = []byte("&quot;")
	escTab  = []byte("&#x9;")
	escNL   = []byte("&#xA;")
	escCR   = []byte("&#xD;")
)

// escapeText escapes text node content per C14N rules:
// & → &amp;  < → &lt;  > → &gt;  \r → &#xD;
func escapeText(w io.Writer, s []byte) error {
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		var esc []byte
		switch r {
		case '&':
			esc = escAmp
		case '<':
			esc = escLT
		case '>':
			esc = escGT
		case '\r':
			esc = escCR
		default:
			i += width
			continue
		}
		if _, err := w.Write(s[last:i]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		i += width
		last = i
	}
	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

// escapeAttrValue escapes attribute values per C14N rules:
// & → &amp;  < → &lt;  " → &quot;  \t → &#x9;  \n → &#xA;  \r → &#xD;
func escapeAttrValue(w io.Writer, s []byte) error {
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		var esc []byte
		switch r {
		case '&':
			esc = escAmp
		case '<':
			esc = escLT
		case '"':
			esc = escQuot
		case '\t':
			esc = escTab
		case '\n':
			esc = escNL
		case '\r':
			esc = escCR
		default:
			i += width
			continue
		}
		if _, err := w.Write(s[last:i]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		i += width
		last = i
	}
	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

// escapePIOrComment escapes processing instruction or comment content:
// \r → &#xD;
func escapePIOrComment(w io.Writer, s []byte) error {
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' {
			if _, err := w.Write(s[last:i]); err != nil {
				return err
			}
			if _, err := w.Write(escCR); err != nil {
				return err
			}
			last = i + 1
		}
	}
	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

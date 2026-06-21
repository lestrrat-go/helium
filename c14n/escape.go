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

// escapeRunes walks s as UTF-8 and writes it to w, replacing any rune for which
// repl returns a non-nil byte slice with that slice. Runs of unreplaced bytes
// are written verbatim in a single Write.
func escapeRunes(w io.Writer, s []byte, repl func(rune) []byte) error {
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		esc := repl(r)
		if esc == nil {
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

// escapeText escapes text node content per C14N rules:
// & → &amp;  < → &lt;  > → &gt;  \r → &#xD;
func escapeText(w io.Writer, s []byte) error {
	return escapeRunes(w, s, func(r rune) []byte {
		switch r {
		case '&':
			return escAmp
		case '<':
			return escLT
		case '>':
			return escGT
		case '\r':
			return escCR
		}
		return nil
	})
}

// escapeAttrValue escapes attribute values per C14N rules:
// & → &amp;  < → &lt;  " → &quot;  \t → &#x9;  \n → &#xA;  \r → &#xD;
func escapeAttrValue(w io.Writer, s []byte) error {
	return escapeRunes(w, s, func(r rune) []byte {
		switch r {
		case '&':
			return escAmp
		case '<':
			return escLT
		case '"':
			return escQuot
		case '\t':
			return escTab
		case '\n':
			return escNL
		case '\r':
			return escCR
		}
		return nil
	})
}

// escapePIOrComment escapes processing instruction or comment content:
// \r → &#xD;
func escapePIOrComment(w io.Writer, s []byte) error {
	last := 0
	for i := range s {
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

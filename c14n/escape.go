package c14n

import "io"

var (
	escAmp  = []byte("&amp;")
	escLT   = []byte("&lt;")
	escGT   = []byte("&gt;")
	escQuot = []byte("&quot;")
	escTab  = []byte("&#x9;")
	escNL   = []byte("&#xA;")
	escCR   = []byte("&#xD;")
)

// escapeTable maps a byte to its replacement, or nil to copy the byte through.
type escapeTable [256][]byte

// textEscapes are the C14N text-node replacements:
// & → &amp;  < → &lt;  > → &gt;  \r → &#xD;
var textEscapes = escapeTable{
	'&':  escAmp,
	'<':  escLT,
	'>':  escGT,
	'\r': escCR,
}

// attrEscapes are the C14N attribute-value replacements:
// & → &amp;  < → &lt;  " → &quot;  \t → &#x9;  \n → &#xA;  \r → &#xD;
var attrEscapes = escapeTable{
	'&':  escAmp,
	'<':  escLT,
	'"':  escQuot,
	'\t': escTab,
	'\n': escNL,
	'\r': escCR,
}

// escapeBytes writes s to w, replacing each byte that has an entry in table.
// Runs of unreplaced bytes are written verbatim in a single Write. Every
// replaced character is ASCII and every byte of a multi-byte UTF-8 sequence is
// at least 0x80, so a byte scan replaces exactly the characters a rune scan
// would, and invalid UTF-8 passes through unchanged.
func escapeBytes(w io.Writer, s []byte, table *escapeTable) error {
	last := 0
	for i, b := range s {
		esc := table[b]
		if esc == nil {
			continue
		}
		if _, err := w.Write(s[last:i]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i + 1
	}
	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

// escapeText escapes text node content per C14N rules:
// & → &amp;  < → &lt;  > → &gt;  \r → &#xD;
func escapeText(w io.Writer, s []byte) error {
	return escapeBytes(w, s, &textEscapes)
}

// escapeAttrValue escapes attribute values per C14N rules:
// & → &amp;  < → &lt;  " → &quot;  \t → &#x9;  \n → &#xA;  \r → &#xD;
func escapeAttrValue(w io.Writer, s []byte) error {
	return escapeBytes(w, s, &attrEscapes)
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

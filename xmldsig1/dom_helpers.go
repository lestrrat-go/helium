package xmldsig1

import (
	"encoding/base64"
	"strings"
)

// decodeBase64 decodes the base64 text of an XML Signature field. XML Signature
// base64 fields (SignatureValue, DigestValue, X509Certificate, key-value
// components, ...) are typed xs:base64Binary, whose lexical space permits
// interspersed XML whitespace; real-world signers line-wrap and indent base64.
// Go's base64 decoder happens to skip CR/LF but rejects space and tab, so all
// four XML whitespace characters (space, tab, CR, LF) are stripped before
// decoding. No other characters are removed, so invalid base64 still fails.
func decodeBase64(text string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(stripXMLWhitespace(text))
}

// stripXMLWhitespace removes the four XML whitespace characters (space 0x20,
// tab 0x09, CR 0x0D, LF 0x0A) from s.
func stripXMLWhitespace(s string) string {
	if !strings.ContainsAny(s, " \t\r\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := range len(s) {
		switch c := s[i]; c {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

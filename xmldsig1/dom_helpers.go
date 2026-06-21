package xmldsig1

import (
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// decodeBase64 decodes the xs:base64Binary text of an XML Signature field
// (SignatureValue, DigestValue, X509Certificate, key-value components, ...),
// stripping interspersed XML whitespace from line-wrapped/indented base64.
func decodeBase64(text string) ([]byte, error) {
	return xmlbase64.DecodeString(text)
}

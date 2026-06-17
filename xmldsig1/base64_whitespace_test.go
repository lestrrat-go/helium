package xmldsig1_test

import (
	"regexp"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// wrapElementBase64 line-wraps the text content of the named element at the
// given column using the full set of XML whitespace separators (CR, LF, tab,
// space), mimicking real-world signers that pretty-print/indent base64. XSD
// base64Binary permits interspersed XML whitespace, so a verifier must tolerate
// all of these. Go's base64 decoder skips CR/LF but rejects space and tab, so
// spaces/tabs are what actually exercise the strip-before-decode fix.
func wrapElementBase64(t *testing.T, xml, tag string, col int) string {
	t.Helper()
	re := regexp.MustCompile(`(<(?:\w+:)?` + tag + `[^>]*>)([^<]+)(</(?:\w+:)?` + tag + `>)`)
	loc := re.FindStringSubmatchIndex(xml)
	require.NotNil(t, loc, "expected to find <%s> element in serialized signature", tag)
	body := xml[loc[4]:loc[5]]
	require.NotContains(t, body, " ", "test setup: %s body should be single-line before wrapping", tag)

	// Rotate through all XML whitespace so the fix must handle every variant.
	seps := []string{"\n", "  \t", "\r\n\t", " "}
	var sb strings.Builder
	sepIdx := 0
	for i, r := range body {
		if i > 0 && i%col == 0 {
			sb.WriteString(seps[sepIdx%len(seps)])
			sepIdx++
		}
		sb.WriteRune(r)
	}
	wrapped := sb.String()
	require.Contains(t, wrapped, " ", "test setup: wrapping should introduce a space into %s", tag)
	return re.ReplaceAllString(xml, "${1}"+wrapped+"${3}")
}

// TestVerifyLineWrappedSignatureValue ensures that a line-wrapped
// SignatureValue (valid xs:base64Binary with interspersed whitespace) still
// verifies rather than failing to base64-decode. SignatureValue lives outside
// SignedInfo, so wrapping it post-signing does not disturb the signed bytes.
func TestVerifyLineWrappedSignatureValue(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	xml, err := helium.WriteString(doc)
	require.NoError(t, err)

	xml = wrapElementBase64(t, xml, "SignatureValue", 64)

	doc2 := mustParseXML(t, xml)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.NoError(t, err)
}

// TestVerifyLineWrappedX509Certificate ensures a line-wrapped
// X509Certificate in KeyInfo still decodes and resolves the key.
func TestVerifyLineWrappedX509Certificate(t *testing.T) {
	key := generateRSAKey(t)
	cert := generateSelfSignedCert(t, key)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(xmldsig1.X509DataKeyInfo(cert))

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	xml, err := helium.WriteString(doc)
	require.NoError(t, err)

	xml = wrapElementBase64(t, xml, "X509Certificate", 64)

	doc2 := mustParseXML(t, xml)

	verifier := xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert))
	_, err = verifier.Verify(t.Context(), doc2)
	require.NoError(t, err)
}

// TestVerifyLineWrappedRSAKeyValue ensures line-wrapped Modulus/Exponent in
// an RSAKeyValue still decode.
func TestVerifyLineWrappedRSAKeyValue(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	xml, err := helium.WriteString(doc)
	require.NoError(t, err)

	xml = wrapElementBase64(t, xml, "Modulus", 64)

	doc2 := mustParseXML(t, xml)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.NoError(t, err)
}

package xmldsig1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestNewEnvelopedReferenceByID signs a nested element by its id and asserts
// the signature covers that nested element, NOT the document root. This is the
// SAML element-signature shape (URI="#AssertionID") that NewEnvelopedReference
// (empty URI, whole-document coverage) cannot express.
func TestNewEnvelopedReferenceByID(t *testing.T) {
	// A Response envelope carrying a nested Assertion. Only the Assertion is
	// signed, targeted by its ID.
	src := `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_resp1" Version="2.0"><saml:Assertion ID="_assertion1" Version="2.0"><saml:Issuer>https://idp.example.com</saml:Issuer></saml:Assertion></samlp:Response>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, src)

	root := doc.DocumentElement()
	assertion := findElementByLocalName(root, "Assertion")
	require.NotNil(t, assertion)
	require.NotEqual(t, root, assertion, "test setup: assertion must be a nested element, not the root")

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReferenceByID("_assertion1"))
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, assertion, key))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	result, err := verifier.Verify(t.Context(), doc)
	require.NoError(t, err)

	require.True(t, result.Covers(assertion), "signature must cover the nested Assertion")
	require.False(t, result.Covers(root), "signature must NOT cover the document root")
	require.Equal(t, assertion, result.SignedElement("#_assertion1"))
}

// TestNewEnvelopedReferenceCoversDocumentRoot pins the whole-document coverage
// of NewEnvelopedReference (empty URI): even when the Signature is inserted
// into a nested parent, the reference resolves to the document element.
func TestNewEnvelopedReferenceCoversDocumentRoot(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	root := doc.DocumentElement()
	subject := findElementByLocalName(root, "Subject")
	require.NotNil(t, subject)
	require.NotEqual(t, root, subject, "test setup: subject must be a nested element")

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, subject, key))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	result, err := verifier.Verify(t.Context(), doc)
	require.NoError(t, err)

	require.True(t, result.Covers(root), "empty-URI reference must cover the document root")
	require.Equal(t, root, result.SignedElement(""))
}

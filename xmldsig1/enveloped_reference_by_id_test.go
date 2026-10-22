package xmldsig1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestNewEnvelopedReferenceByID signs a nested element by its id and asserts
// the Reference resolves to that nested element, NOT the document root. The
// enveloped transform excludes the Signature subtree, so it does not claim
// whole-element coverage.
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

	require.False(t, result.Covers(assertion), "enveloped transform excludes part of the Assertion")
	require.False(t, result.Covers(root), "signature must NOT cover the document root")
	require.Nil(t, result.SignedElement("#_assertion1"))
	require.Same(t, assertion, result.References[0].Element)
}

// TestNewEnvelopedReferenceExposesDocumentRoot pins the resolved target for an
// empty-URI Reference while declining whole-element coverage after filtering.
func TestNewEnvelopedReferenceExposesDocumentRoot(t *testing.T) {
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

	require.False(t, result.Covers(root), "enveloped transform excludes the Signature subtree")
	require.Nil(t, result.SignedElement(""))
	require.Same(t, root, result.References[0].Element)
}

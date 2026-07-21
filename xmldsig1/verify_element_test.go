package xmldsig1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestVerifyElementNilSignature confirms VerifyElement rejects a nil Signature
// element with a typed error instead of panicking on a nil dereference. Unlike
// Verify, VerifyElement takes the target element from the caller, so a caller
// that located nothing (nil) must get an error back, matching the validation
// findSignatureElements performs for the Verify path.
func TestVerifyElementNilSignature(t *testing.T) {
	doc := mustParseXML(t, samlAssertion)
	key := generateRSAKey(t)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))

	_, err := verifier.VerifyElement(t.Context(), doc, nil)
	require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
}

// TestVerifyElementRejectsNonSignature confirms VerifyElement refuses an element
// that is not a ds:Signature in the core XML-Signature namespace, matching the
// local-name + namespace gate findSignatureElements applies for Verify.
func TestVerifyElementRejectsNonSignature(t *testing.T) {
	key := generateRSAKey(t)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))

	t.Run("plain element", func(t *testing.T) {
		doc := mustParseXML(t, `<foo><bar/></foo>`)
		_, err := verifier.VerifyElement(t.Context(), doc, doc.DocumentElement())
		require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
	})

	t.Run("signature in wrong namespace", func(t *testing.T) {
		doc := mustParseXML(t, `<Signature xmlns="http://example.com/not-dsig"><SignedInfo/></Signature>`)
		_, err := verifier.VerifyElement(t.Context(), doc, doc.DocumentElement())
		require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
	})
}

// TestVerifyElementValid confirms a genuine ds:Signature element still verifies
// through VerifyElement once the nil / non-Signature guards are in place.
func TestVerifyElementValid(t *testing.T) {
	xml := `<root><data Id="mydata">Hello</data></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	ref := xmldsig1.ReferenceConfig{
		URI:             "#mydata",
		DigestAlgorithm: xmldsig1.DigestSHA256,
		Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
	}

	sigElem, err := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(ref).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	result, err := verifier.VerifyElement(t.Context(), doc, sigElem)
	require.NoError(t, err)
	require.NotNil(t, result)
}

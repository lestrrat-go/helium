package xmldsig1_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// signTwoReferenceDoc signs samlAssertion with two whole-document enveloped
// References, producing a validly-signed document that carries more than one
// Reference so the MaxReferences cap can be exercised end to end.
func signTwoReferenceDoc(t *testing.T, key any) *helium.Document {
	t.Helper()
	doc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))
	return doc
}

// TestVerifyResourceLimits covers the opt-in parse-time resource caps that bound
// the decode/parse work an attacker-controlled Signature can force before the
// SignatureValue is checked.
func TestVerifyResourceLimits(t *testing.T) {
	// MaxReferences: a document declaring more References than the cap is
	// rejected with ErrResourceLimitExceeded before any Reference is digested,
	// while the same document verifies under the default (generous) cap.
	t.Run("max references", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := signTwoReferenceDoc(t, key)

		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxReferences(1).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)

		// Default cap leaves the two-Reference document verifying normally.
		_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)

		// A negative cap disables the check entirely.
		_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxReferences(-1).
			Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// MaxDecodedBytes: a running total of base64-decoded bytes (DigestValue,
	// SignatureValue, X509Certificate) over the cap is rejected. A one-byte cap
	// trips on the first DigestValue decode; the default cap is unaffected.
	t.Run("max decoded bytes", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxDecodedBytes(1).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)

		_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// MaxKeyInfoEntries: a KeyInfo carrying more entries than the cap is rejected.
	// A KeyInfo with two X509Certificate children exceeds a cap of 1; the default
	// cap leaves it verifying.
	t.Run("max key info entries", func(t *testing.T) {
		key := generateRSAKey(t)
		cert := generateSelfSignedCert(t, key)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference()).
			KeyInfo(xmldsig1.X509DataKeyInfo(cert, cert))
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		_, err := xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert)).
			MaxKeyInfoEntries(1).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)

		_, err = xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

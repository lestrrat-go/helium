package xmldsig1_test

import (
	"context"
	"crypto"
	"crypto/elliptic"
	"crypto/rsa"
	"io"
	"regexp"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// hsmSigner is an opaque crypto.Signer wrapper simulating an HSM/KMS/PKCS#11
// key. It is deliberately NOT one of the concrete key types the fast path
// accepts (*rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey), so signing
// through it exercises the crypto.Signer fallback.
type hsmSigner struct {
	inner crypto.Signer
}

func (s hsmSigner) Public() crypto.PublicKey { return s.inner.Public() }

func (s hsmSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.inner.Sign(rand, digest, opts)
}

var sigValueRe = regexp.MustCompile(`(?s)<ds:SignatureValue[^>]*>(.*?)</ds:SignatureValue>`)

// signatureValue serializes doc and extracts the ds:SignatureValue text.
func signatureValue(t *testing.T, doc *helium.Document) string {
	t.Helper()
	xml, err := helium.WriteString(doc)
	require.NoError(t, err)
	m := sigValueRe.FindStringSubmatch(xml)
	require.Len(t, m, 2)
	return m[1]
}

// TestSignWithCryptoSigner confirms a caller-supplied crypto.Signer (an opaque
// HSM/KMS-style key that is not one of the concrete fast-path types) can sign,
// and the resulting signature verifies with the matching public key.
func TestSignWithCryptoSigner(t *testing.T) {
	t.Run("rsa-sha256", func(t *testing.T) {
		key := generateRSAKey(t)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())

		doc := mustParseXML(t, samlAssertion)
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), hsmSigner{inner: key}))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)

		// PKCS1v15 is deterministic: the crypto.Signer path must produce the
		// byte-identical SignatureValue as the concrete-key path.
		concrete := mustParseXML(t, samlAssertion)
		require.NoError(t, signer.SignEnveloped(t.Context(), concrete, concrete.DocumentElement(), key))
		require.Equal(t, signatureValue(t, concrete), signatureValue(t, doc))
	})

	t.Run("ecdsa-sha256", func(t *testing.T) {
		key := generateECDSAKey(t, elliptic.P256())
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgECDSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())

		doc := mustParseXML(t, samlAssertion)
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), hsmSigner{inner: key}))

		// ECDSA signing is randomized, so assert the signature VERIFIES rather
		// than byte-equality against a fixed vector.
		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("ecdsa-sha384", func(t *testing.T) {
		key := generateECDSAKey(t, elliptic.P384())
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgECDSASHA384).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: xmldsig1.DigestSHA384,
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})

		doc := mustParseXML(t, samlAssertion)
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), hsmSigner{inner: key}))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("ed25519", func(t *testing.T) {
		key := generateEd25519Key(t)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgEd25519).
			Reference(xmldsig1.NewEnvelopedReference())

		doc := mustParseXML(t, samlAssertion)
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), hsmSigner{inner: key}))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(key.Public()))
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)

		// Ed25519 is deterministic: byte-identical SignatureValue expected.
		concrete := mustParseXML(t, samlAssertion)
		require.NoError(t, signer.SignEnveloped(t.Context(), concrete, concrete.DocumentElement(), key))
		require.Equal(t, signatureValue(t, concrete), signatureValue(t, doc))
	})
}

// TestSignDetachedWithCryptoSigner confirms the crypto.Signer fallback also
// works on the detached-signature path.
func TestSignDetachedWithCryptoSigner(t *testing.T) {
	key := generateRSAKey(t)
	xml := `<root><data Id="mydata">Hello</data></root>`
	doc := mustParseXML(t, xml)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             refURIMyData,
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})

	sigElem, err := signer.SignDetached(t.Context(), doc, hsmSigner{inner: key})
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestSignCryptoSignerKeyInfo confirms RSAKeyValueKeyInfo derives the
// RSAKeyValue from an opaque crypto.Signer's public key (via Public()), so a
// signature by an HSM/KMS-style RSA key completes with a correct KeyInfo. The
// emitted SignatureValue and KeyInfo must match the concrete-*rsa.PrivateKey
// output, and the signature must verify against the KeyInfo's RSAKeyValue.
func TestSignCryptoSignerKeyInfo(t *testing.T) {
	key := generateRSAKey(t)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

	doc := mustParseXML(t, samlAssertion)
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), hsmSigner{inner: key}))

	// The opaque-signer output must be byte-identical to the concrete-key output
	// (PKCS1v15 is deterministic and the KeyInfo derives from the same public key).
	concrete := mustParseXML(t, samlAssertion)
	require.NoError(t, signer.SignEnveloped(t.Context(), concrete, concrete.DocumentElement(), key))
	require.Equal(t, signatureValue(t, concrete), signatureValue(t, doc))

	concreteXML, err := helium.WriteString(concrete)
	require.NoError(t, err)
	opaqueXML, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Equal(t, concreteXML, opaqueXML)

	// Resolve the verification key straight out of the emitted RSAKeyValue to
	// prove the KeyInfo carries the correct modulus/exponent and the signature
	// verifies against it.
	ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
		require.NotNil(t, ki.RSAKeyValue)
		require.Equal(t, key.N, ki.RSAKeyValue.Modulus)
		require.Equal(t, key.E, ki.RSAKeyValue.Exponent)
		return &rsa.PublicKey{N: ki.RSAKeyValue.Modulus, E: ki.RSAKeyValue.Exponent}, nil
	})
	_, err = xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestSignCryptoSignerWrongAlgorithm confirms a crypto.Signer whose public key
// type does not match the requested algorithm is still rejected with
// ErrKeyMismatch.
func TestSignCryptoSignerWrongAlgorithm(t *testing.T) {
	// An ECDSA-backed signer used for an RSA signature algorithm.
	key := generateECDSAKey(t, elliptic.P256())
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())

	doc := mustParseXML(t, samlAssertion)
	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), hsmSigner{inner: key})
	require.ErrorIs(t, err, xmldsig1.ErrKeyMismatch)
}

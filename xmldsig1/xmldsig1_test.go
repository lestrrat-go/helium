package xmldsig1_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

func mustParseXML(t *testing.T, xml string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	return doc
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func generateECDSAKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	return key
}

func generateEd25519Key(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return priv
}

func generateSelfSignedCert(t *testing.T, key *rsa.PrivateKey) *x509.Certificate {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	return cert
}

const samlAssertion = `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc123" IssueInstant="2024-01-01T00:00:00Z" Version="2.0"><saml:Issuer>https://idp.example.com</saml:Issuer><saml:Subject><saml:NameID>user@example.com</saml:NameID></saml:Subject></saml:Assertion>`

func TestSignVerifyRoundTrip(t *testing.T) {
	t.Run("rsa-sha256", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference()).
			KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.NoError(t, err)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("rsa-sha1", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := signRSASHA1Doc(t, key)

		// SHA-1 verification must be opted into; the default verifier rejects it.
		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).AllowSHA1(true)
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("ecdsa-sha256", func(t *testing.T) {
		key := generateECDSAKey(t, elliptic.P256())
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgECDSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())

		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.NoError(t, err)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("ecdsa-sha384", func(t *testing.T) {
		key := generateECDSAKey(t, elliptic.P384())
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgECDSASHA384).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: xmldsig1.DigestSHA384,
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})

		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.NoError(t, err)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	t.Run("hmac-sha256", func(t *testing.T) {
		secret := make([]byte, 32)
		_, err := rand.Read(secret)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgHMACSHA256).
			Reference(xmldsig1.NewEnvelopedReference())

		err = signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), secret)
		require.NoError(t, err)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(secret))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

// signRSASHA1Doc signs samlAssertion with rsa-sha1 + sha1 digest, opting in to
// SHA-1 on the signer (required since SHA-1 is rejected by default).
func signRSASHA1Doc(t *testing.T, key *rsa.PrivateKey) *helium.Document {
	t.Helper()
	doc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		AllowSHA1(true).
		SignatureAlgorithm(xmldsig1.AlgRSASHA1).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "",
			DigestAlgorithm: xmldsig1.DigestSHA1,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
		})
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))
	return doc
}

// TestDefaultPolicy covers the default SHA-1 rejection policy and its opt-in.
func TestDefaultPolicy(t *testing.T) {
	// TestSignSHA1RejectedByDefault confirms that signing with SHA-1 is rejected
	// unless Signer.AllowSHA1(true) is set.
	t.Run("sign sha1 rejected by default", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA1).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: xmldsig1.DigestSHA1,
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})

		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)
	})

	// TestVerifySHA1RejectedByDefault confirms an rsa-sha1 + sha1 signature is
	// rejected by the default verifier with ErrWeakAlgorithm.
	t.Run("verify sha1 rejected by default", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := signRSASHA1Doc(t, key)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err := verifier.Verify(t.Context(), doc)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)
	})

	// TestVerifySHA1AcceptedWithOptIn confirms the same signature verifies once
	// SHA-1 is opted in on the verifier.
	t.Run("verify sha1 accepted with opt-in", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := signRSASHA1Doc(t, key)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).AllowSHA1(true)
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// TestVerifySHA256AcceptedByDefault confirms strong algorithms still verify
	// without any opt-in.
	t.Run("verify sha256 accepted by default", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

func TestSignVerifyRoundTripEd25519(t *testing.T) {
	key := generateEd25519Key(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgEd25519).
		Reference(xmldsig1.NewEnvelopedReference())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(key.Public()))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

func TestSignVerifyWithX509CertKeySource(t *testing.T) {
	key := generateRSAKey(t)
	cert := generateSelfSignedCert(t, key)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(xmldsig1.X509DataKeyInfo(cert))

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.X509CertKeySource(cert))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

func TestVerifyTamperedDocument(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	// Serialize, tamper, re-parse.
	xml, err := helium.WriteString(doc)
	require.NoError(t, err)
	tampered := strings.Replace(xml, "user@example.com", "attacker@evil.com", 1)
	doc2 := mustParseXML(t, tampered)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc2)
	require.Error(t, err)
}

func TestVerifyWrongKey(t *testing.T) {
	key1 := generateRSAKey(t)
	key2 := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key1)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key2.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.Error(t, err)
}

func TestVerifyNoSignature(t *testing.T) {
	doc := mustParseXML(t, samlAssertion)
	key := generateRSAKey(t)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err := verifier.Verify(t.Context(), doc)
	require.ErrorIs(t, err, xmldsig1.ErrSignatureNotFound)
}

func TestSignNoReferences(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256)

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.ErrorIs(t, err, xmldsig1.ErrNoReferences)
}

func TestSignNoAlgorithm(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		Reference(xmldsig1.NewEnvelopedReference())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.Error(t, err)
}

func TestSignerImmutability(t *testing.T) {
	s1 := xmldsig1.NewSigner().SignatureAlgorithm(xmldsig1.AlgRSASHA256)
	s2 := s1.Reference(xmldsig1.NewEnvelopedReference())

	// s1 should not have any references; s2 should have one.
	key := generateRSAKey(t)
	doc1 := mustParseXML(t, samlAssertion)
	doc2 := mustParseXML(t, samlAssertion)

	err := s1.SignEnveloped(t.Context(), doc1, doc1.DocumentElement(), key)
	require.ErrorIs(t, err, xmldsig1.ErrNoReferences)

	err = s2.SignEnveloped(t.Context(), doc2, doc2.DocumentElement(), key)
	require.NoError(t, err)
}

// TestZeroValueSignerReturnsErrNoReferences guards the three sign terminals
// against a nil cfg: a zero-value Signer{} (built directly, not via NewSigner)
// has no references configured, so each terminal must return ErrNoReferences
// exactly as NewSigner() with no references does — never panic on a nil cfg.
func TestZeroValueSignerReturnsErrNoReferences(t *testing.T) {
	key := generateRSAKey(t)

	t.Run("SignEnveloped", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		err := xmldsig1.Signer{}.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.ErrorIs(t, err, xmldsig1.ErrNoReferences)
	})

	t.Run("SignEnveloping", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		payload, err := doc.CreateElement("Payload")
		require.NoError(t, err)
		_, err = xmldsig1.Signer{}.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.ErrorIs(t, err, xmldsig1.ErrNoReferences)
	})

	t.Run("SignDetached", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := xmldsig1.Signer{}.SignDetached(t.Context(), doc, key)
		require.ErrorIs(t, err, xmldsig1.ErrNoReferences)
	})
}

// TestVerifyAcceptsEnvelopedExcC14N guards against the unsupported-transform
// rejection over-rejecting: a normal enveloped + exclusive-c14n signature must
// still verify (both transform URIs must pass the new default-arm guard).
func TestVerifyAcceptsEnvelopedExcC14N(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
		})

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

func TestSignVerifyWithFragmentReference(t *testing.T) {
	xml := `<root><data Id="mydata">Hello</data></root>`
	key := generateRSAKey(t)
	doc := mustParseXML(t, xml)

	ref := xmldsig1.ReferenceConfig{
		URI:             "#mydata",
		DigestAlgorithm: xmldsig1.DigestSHA256,
		Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
	}

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(ref)

	sigElem, err := signer.SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NotNil(t, sigElem)

	err = doc.DocumentElement().AddChild(sigElem)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

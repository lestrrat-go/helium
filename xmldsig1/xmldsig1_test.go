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

func TestSignVerifyRoundTripRSASHA256(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

func TestSignVerifyRoundTripRSASHA1(t *testing.T) {
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
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

func TestSignVerifyRoundTripECDSASHA256(t *testing.T) {
	key := generateECDSAKey(t, elliptic.P256())
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgECDSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())

	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

func TestSignVerifyRoundTripECDSASHA384(t *testing.T) {
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
	err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

func TestSignVerifyRoundTripHMACSHA256(t *testing.T) {
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
	err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
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
	err = verifier.Verify(t.Context(), doc)
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
	err = verifier.Verify(t.Context(), doc)
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
	err = verifier.Verify(t.Context(), doc2)
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
	err = verifier.Verify(t.Context(), doc)
	require.Error(t, err)
}

func TestVerifyNoSignature(t *testing.T) {
	doc := mustParseXML(t, samlAssertion)
	key := generateRSAKey(t)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	err := verifier.Verify(t.Context(), doc)
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
	err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

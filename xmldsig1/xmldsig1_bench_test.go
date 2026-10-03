package xmldsig1_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/xmldsig1"
)

// benchSAMLAssertion is a ~1KB SAML 2.0 assertion, the typical size of a
// document signed in a web single sign-on exchange.
const benchSAMLAssertion = `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_d71a3a8e9fcc45c9e9d248ef7049393fc8f04e5f75" IssueInstant="2024-01-01T00:00:00Z" Version="2.0">
  <saml:Issuer>https://idp.example.com/metadata</saml:Issuer>
  <saml:Subject>
    <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">user@example.com</saml:NameID>
    <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml:SubjectConfirmationData NotOnOrAfter="2024-01-01T00:05:00Z" Recipient="https://sp.example.com/acs" InResponseTo="_req-4fe1b2"/>
    </saml:SubjectConfirmation>
  </saml:Subject>
  <saml:Conditions NotBefore="2024-01-01T00:00:00Z" NotOnOrAfter="2024-01-01T00:05:00Z">
    <saml:AudienceRestriction>
      <saml:Audience>https://sp.example.com/metadata</saml:Audience>
    </saml:AudienceRestriction>
  </saml:Conditions>
  <saml:AuthnStatement AuthnInstant="2024-01-01T00:00:00Z" SessionIndex="_session-8c2d">
    <saml:AuthnContext>
      <saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport</saml:AuthnContextClassRef>
    </saml:AuthnContext>
  </saml:AuthnStatement>
  <saml:AttributeStatement>
    <saml:Attribute Name="email"><saml:AttributeValue>user@example.com</saml:AttributeValue></saml:Attribute>
    <saml:Attribute Name="groups"><saml:AttributeValue>engineering</saml:AttributeValue></saml:Attribute>
  </saml:AttributeStatement>
</saml:Assertion>`

// benchDocument is one unsigned input document for the sign/verify benchmarks.
type benchDocument struct {
	name string
	data []byte
	doc  *helium.Document
}

// benchKey is one signature algorithm and its key pair for the benchmarks.
type benchKey struct {
	name string
	alg  string
	priv any
	pub  any
}

// loadBenchDocuments parses the benchmark inputs once: the inline ~1KB SAML
// assertion and the ~287KB nvdcve_0.xml fixture, which shows how
// canonicalization and digesting scale with document size.
func loadBenchDocuments(b *testing.B) []benchDocument {
	b.Helper()
	nvd, err := os.ReadFile(filepath.Join(heliumtest.RepoRoot(), "testdata/libxml2-compat/schemas/test/nvdcve_0.xml"))
	if err != nil {
		b.Fatal(err)
	}
	docs := []benchDocument{
		{name: "saml_1KB", data: []byte(benchSAMLAssertion)},
		{name: "nvdcve_287KB", data: nvd},
	}
	for i := range docs {
		doc, err := helium.NewParser().Parse(b.Context(), docs[i].data)
		if err != nil {
			b.Fatalf("parse %s: %s", docs[i].name, err)
		}
		docs[i].doc = doc
	}
	return docs
}

// generateBenchKeys generates one RSA-2048 and one ECDSA P-256 key pair, so
// RSA and EC signing costs are visible side by side.
func generateBenchKeys(b *testing.B) []benchKey {
	b.Helper()
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	return []benchKey{
		{name: "rsa-sha256", alg: xmldsig1.AlgRSASHA256, priv: rsaKey, pub: &rsaKey.PublicKey},
		{name: "ecdsa-p256-sha256", alg: xmldsig1.AlgECDSASHA256, priv: ecKey, pub: &ecKey.PublicKey},
	}
}

// newBenchSigner returns the common enveloped-signature configuration: the
// NewSigner default Exclusive C14N for SignedInfo and a whole-document
// Reference (URI="") with the enveloped-signature transform, Exclusive C14N,
// and a SHA-256 digest.
func newBenchSigner(alg string) xmldsig1.Signer {
	return xmldsig1.NewSigner().
		SignatureAlgorithm(alg).
		Reference(xmldsig1.NewEnvelopedReference())
}

// signBenchCopy deep-copies src and signs the copy under its document element.
func signBenchCopy(b *testing.B, signer xmldsig1.Signer, src *helium.Document, key any) *helium.Document {
	b.Helper()
	doc, err := helium.CopyDoc(src)
	if err != nil {
		b.Fatal(err)
	}
	if err := signer.SignEnveloped(b.Context(), doc, doc.DocumentElement(), key); err != nil {
		b.Fatal(err)
	}
	return doc
}

// BenchmarkSignEnveloped measures Signer.SignEnveloped per document size and
// signature algorithm. Signing inserts a Signature into the document, so every
// iteration signs a fresh helium.CopyDoc of the parsed input. The copy runs
// with the timer stopped, so ns/op and allocs/op cover signing only. SetBytes
// uses the unsigned input size.
func BenchmarkSignEnveloped(b *testing.B) {
	docs := loadBenchDocuments(b)
	keys := generateBenchKeys(b)
	for _, d := range docs {
		for _, k := range keys {
			b.Run(d.name+"/"+k.name, func(b *testing.B) {
				runSignEnveloped(b, newBenchSigner(k.alg), d, k.priv)
			})
		}
	}
}

func runSignEnveloped(b *testing.B, signer xmldsig1.Signer, d benchDocument, key any) {
	ctx := b.Context()
	b.SetBytes(int64(len(d.data)))
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		doc, err := helium.CopyDoc(d.doc)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := signer.SignEnveloped(ctx, doc, doc.DocumentElement(), key); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVerify measures Verifier.Verify per document size and signature
// algorithm. Each case signs one copy of the input outside the timed loop and
// verifies that same document every iteration; Verify does not mutate the
// document (TestVerifyEnveloped/"does not mutate dom"). SetBytes uses the
// unsigned input size, matching BenchmarkSignEnveloped.
func BenchmarkVerify(b *testing.B) {
	docs := loadBenchDocuments(b)
	keys := generateBenchKeys(b)
	for _, d := range docs {
		for _, k := range keys {
			b.Run(d.name+"/"+k.name, func(b *testing.B) {
				signed := signBenchCopy(b, newBenchSigner(k.alg), d.doc, k.priv)
				verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(k.pub))
				runVerify(b, verifier, signed, len(d.data))
			})
		}
	}
}

func runVerify(b *testing.B, verifier xmldsig1.Verifier, signed *helium.Document, size int) {
	ctx := b.Context()
	b.SetBytes(int64(size))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := verifier.Verify(ctx, signed); err != nil {
			b.Fatal(err)
		}
	}
}

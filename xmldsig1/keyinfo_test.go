package xmldsig1_test

import (
	"context"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestResolveKey covers resolving the verification key from parsed KeyInfo.
func TestResolveKey(t *testing.T) {
	// rsa key value signs with RSAKeyValueKeyInfo, then resolves the verification
	// key out of the parsed KeyInfoData.RSAKeyValue. This drives parseKeyInfo ->
	// parseKeyValue -> parseRSAKeyValue and KeyInfoData wiring.
	t.Run("rsa key value", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference()).
			KeyInfo(xmldsig1.RSAKeyValueKeyInfo())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
			require.NotNil(t, ki)
			require.NotNil(t, ki.RSAKeyValue)
			return &rsa.PublicKey{
				N: ki.RSAKeyValue.Modulus,
				E: ki.RSAKeyValue.Exponent,
			}, nil
		})
		verifier := xmldsig1.NewVerifier(ks)
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// x509 data signs with X509DataKeyInfo, then reads the parsed certificate from
	// KeyInfoData.X509Certificates. Drives parseX509Data.
	t.Run("x509 data", func(t *testing.T) {
		key := generateRSAKey(t)
		cert := generateSelfSignedCert(t, key)
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference()).
			KeyInfo(xmldsig1.X509DataKeyInfo(cert))
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
			require.NotNil(t, ki)
			require.Len(t, ki.X509Certificates, 1)
			return ki.X509Certificates[0].PublicKey, nil
		})
		verifier := xmldsig1.NewVerifier(ks)
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

// TestParseECKeyValue drives parseECKeyValue (P-256 and P-384) and isDSig11NS
// through a verification using a dsig11 ECKeyValue KeyInfo.
func TestParseECKeyValue(t *testing.T) {
	for _, tc := range []struct {
		name     string
		curve    elliptic.Curve
		curveURI string
		alg      string
	}{
		{"p256", elliptic.P256(), "urn:oid:1.2.840.10045.3.1.7", xmldsig1.AlgECDSASHA256},
		{"p384", elliptic.P384(), "urn:oid:1.3.132.0.34", xmldsig1.AlgECDSASHA384},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := generateECDSAKey(t, tc.curve)
			doc := mustParseXML(t, samlAssertion)
			signer := xmldsig1.NewSigner().
				SignatureAlgorithm(tc.alg).
				Reference(xmldsig1.ReferenceConfig{
					URI:             "",
					DigestAlgorithm: dsigDigestFor(tc.alg),
					Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
				})
			require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

			// Inject a dsig11 ECKeyValue KeyInfo into the Signature so the
			// verifier parses it.
			pubBytes := elliptic.Marshal(tc.curve, key.X, key.Y)
			injectECKeyInfo(t, doc, tc.curveURI, pubBytes)

			ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
				require.NotNil(t, ki.ECKeyValue)
				require.NotNil(t, ki.ECKeyValue.X)
				return &key.PublicKey, nil
			})
			verifier := xmldsig1.NewVerifier(ks)
			_, err := verifier.Verify(t.Context(), doc)
			require.NoError(t, err)
		})
	}
}

func dsigDigestFor(alg string) string {
	if alg == xmldsig1.AlgECDSASHA384 {
		return xmldsig1.DigestSHA384
	}
	return xmldsig1.DigestSHA256
}

// injectECKeyInfo appends a <ds:KeyInfo><ds:KeyValue><dsig11:ECKeyValue> ...
// element to the Signature element of doc.
func injectECKeyInfo(t *testing.T, doc *helium.Document, curveURI string, pub []byte) {
	t.Helper()
	const dsig11 = xmldsig1.NamespaceDSig11

	sig := findSig(t, doc)

	keyInfo, err := doc.CreateElement("KeyInfo")
	require.NoError(t, err)
	require.NoError(t, keyInfo.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	keyValue, err := doc.CreateElement("KeyValue")
	require.NoError(t, err)
	require.NoError(t, keyValue.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	require.NoError(t, keyInfo.AddChild(keyValue))

	ec, err := doc.CreateElement("ECKeyValue")
	require.NoError(t, err)
	require.NoError(t, ec.DeclareNamespace("dsig11", dsig11))
	require.NoError(t, ec.SetActiveNamespace("dsig11", dsig11))
	require.NoError(t, keyValue.AddChild(ec))

	nc, err := doc.CreateElement("NamedCurve")
	require.NoError(t, err)
	require.NoError(t, nc.SetActiveNamespace("dsig11", dsig11))
	require.NoError(t, nc.SetLiteralAttribute("URI", curveURI))
	require.NoError(t, ec.AddChild(nc))

	pk, err := doc.CreateElement("PublicKey")
	require.NoError(t, err)
	require.NoError(t, pk.SetActiveNamespace("dsig11", dsig11))
	encoded := base64StdEncode(pub)
	require.NoError(t, pk.AddChild(doc.CreateText([]byte(encoded))))
	require.NoError(t, ec.AddChild(pk))

	require.NoError(t, sig.AddChild(keyInfo))
}

func findSig(t *testing.T, doc *helium.Document) *helium.Element {
	t.Helper()
	var out *helium.Element
	var walk func(helium.Node)
	walk = func(n helium.Node) {
		if e, ok := helium.AsNode[*helium.Element](n); ok {
			if localName(e) == "Signature" {
				out = e
				return
			}
			for c := e.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
	}
	walk(doc.DocumentElement())
	require.NotNil(t, out)
	return out
}

func localName(e *helium.Element) string {
	name := e.Name()
	if _, rest, ok := cut(name, ":"); ok {
		return rest
	}
	return name
}

func cut(s, sep string) (before, after string, found bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}

func base64StdEncode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

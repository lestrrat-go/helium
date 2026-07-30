package xmldsig1_test

import (
	"context"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"strings"
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

// TestX509CertKeySourceNil confirms that a KeySource built from a nil
// certificate resolves to a typed ErrNoKeySource instead of panicking on a nil
// dereference. This is reachable from realistic per-request certificate
// resolution where an unknown issuer yields a nil *x509.Certificate.
func TestX509CertKeySourceNil(t *testing.T) {
	ks := xmldsig1.X509CertKeySource(nil)
	_, err := ks.ResolveKey(t.Context(), nil, "")
	require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
}

// TestX509DataKeyInfoEmpty confirms that X509DataKeyInfo with zero certificates
// fails signing with ErrInvalidKeyInfo rather than silently emitting a
// schema-invalid empty <X509Data>.
func TestX509DataKeyInfoEmpty(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(xmldsig1.X509DataKeyInfo())
	err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
	require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)
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
		{"p521", elliptic.P521(), "urn:oid:1.3.132.0.35", xmldsig1.AlgECDSASHA512},
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
	switch alg {
	case xmldsig1.AlgECDSASHA384:
		return xmldsig1.DigestSHA384
	case xmldsig1.AlgECDSASHA512:
		return xmldsig1.DigestSHA512
	default:
		return xmldsig1.DigestSHA256
	}
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
	require.NoError(t, nc.SetAttribute("URI", curveURI))
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

// signedKeyInfoDoc signs a small document with a whole-document enveloped
// Reference and the given KeyInfo, and returns the serialized result for a case
// to rewrite before reparsing it.
//
// The Reference's enveloped-signature transform removes the whole ds:Signature
// before the digest, and ds:KeyInfo is inside it, so rewriting a KeyInfo value
// leaves a valid signature. Each case below then turns on the KeyInfo parse
// alone: it runs before the key is resolved and before any crypto, so a KeyInfo
// the parse rejects fails the whole verification.
func signedKeyInfoDoc(t *testing.T, key *rsa.PrivateKey, keyInfo xmldsig1.KeyInfoBuilder) string {
	t.Helper()
	doc := mustParseXML(t, `<doc><data>signed content</data></doc>`)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference()).
		KeyInfo(keyInfo)
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))
	serialized, err := helium.WriteString(doc)
	require.NoError(t, err)
	return serialized
}

// valueTextBounds returns the offsets of the text of the first ds:local element
// in serialized.
func valueTextBounds(t *testing.T, serialized, local string) (int, int) {
	t.Helper()
	open := "<ds:" + local + ">"
	openAt := strings.Index(serialized, open)
	require.Positive(t, openAt, "signed document has no ds:%s", local)
	start := openAt + len(open)
	end := strings.Index(serialized, "</ds:"+local+">")
	require.Greater(t, end, start, "ds:%s has no text", local)
	return start, end
}

// commentInsideValue splices a comment into the middle of the text of the first
// ds:local element, which is what a producer annotating a value inside the
// element emits. Concatenating the element's child content would fold the
// comment's own text into the base64 and leave the value undecodable.
func commentInsideValue(t *testing.T, serialized, local string) string {
	t.Helper()
	start, end := valueTextBounds(t, serialized, local)
	mid := start + (end-start)/2
	return serialized[:mid] + "<!--XX-->" + serialized[mid:]
}

// entityWrittenValue rewrites the whole text of the first ds:local element into
// a single reference to an internal-subset entity declaring that same text.
// helium's parser does not substitute entities, so the value arrives as one
// entity-reference child — the shape a document writing its base64 through an
// entity actually has. The value's characters do not change, so the signature is
// still the same signature.
func entityWrittenValue(t *testing.T, serialized, local string) string {
	t.Helper()
	start, end := valueTextBounds(t, serialized, local)
	rootAt := strings.Index(serialized, "<doc>")
	require.Positive(t, rootAt, "signed document has no doc element")
	return serialized[:rootAt] +
		`<!DOCTYPE doc [<!ENTITY val "` + serialized[start:end] + `">]>` +
		serialized[rootAt:start] + "&val;" + serialized[end:]
}

// TestKeyInfoBase64CharacterData covers reading the KeyInfo base64 values as
// character data. A comment inside a value contributes no character information
// item and is skipped instead of being spliced into the base64, and a value
// written through an entity reference still decodes. Each case verifies a real
// signature, because a rejected KeyInfo parse fails the whole verification.
func TestKeyInfoBase64CharacterData(t *testing.T) {
	// commented modulus: the ds:Modulus still decodes to the signing key's
	// modulus, so the RSAKeyValue resolves and the signature verifies.
	t.Run("commented modulus", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, commentInsideValue(t, signedKeyInfoDoc(t, key, xmldsig1.RSAKeyValueKeyInfo()), "Modulus"))

		ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
			require.NotNil(t, ki.RSAKeyValue)
			require.Zero(t, ki.RSAKeyValue.Modulus.Cmp(key.N),
				"the comment must not be spliced into the modulus")
			return &key.PublicKey, nil
		})
		_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// commented X509SKI: the signature is verified with the certificate's key and
	// the SKI is only a selector, so a commented ds:X509SKI must not fail the
	// verification — and must decode to the octets it encodes.
	t.Run("commented X509SKI", func(t *testing.T) {
		key := generateRSAKey(t)
		cert := generateSelfSignedCert(t, key)
		ski := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		encoded := base64StdEncode(ski)

		signed := signedKeyInfoDoc(t, key, xmldsig1.X509DataKeyInfo(cert))
		at := strings.Index(signed, "</ds:X509Data>")
		require.Positive(t, at, "signed document has no ds:X509Data")
		doc := mustParseXML(t, signed[:at]+
			`<ds:X509SKI>`+encoded[:3]+`<!--XX-->`+encoded[3:]+`</ds:X509SKI>`+
			signed[at:])

		ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
			require.Len(t, ki.X509SKIs, 1)
			require.Equal(t, ski, ki.X509SKIs[0], "the comment must not be spliced into the X509SKI")
			return cert.PublicKey, nil
		})
		_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// entity-written modulus: an entity reference stands for character data, so a
	// modulus written as one decodes to the same modulus.
	t.Run("entity-written modulus", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, entityWrittenValue(t, signedKeyInfoDoc(t, key, xmldsig1.RSAKeyValueKeyInfo()), "Modulus"))

		ks := xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
			require.NotNil(t, ki.RSAKeyValue)
			require.Zero(t, ki.RSAKeyValue.Modulus.Cmp(key.N),
				"an entity-written modulus must decode to the same modulus")
			return &key.PublicKey, nil
		})
		_, err := xmldsig1.NewVerifier(ks).Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

// TestX509DataKeyInfoRejectsNilCert proves a nil *x509.Certificate entry in the
// varargs is rejected with a typed ErrInvalidKeyInfo rather than panicking on
// cert.Raw, both when BuildKeyInfo is invoked directly and when it is reached
// through a signing entry point. SignEnveloping rejects it in its preflight,
// before any caller content is moved into the <Object>.
func TestX509DataKeyInfoRejectsNilCert(t *testing.T) {
	key := generateRSAKey(t)
	cert := generateSelfSignedCert(t, key)

	t.Run("BuildKeyInfo", func(t *testing.T) {
		doc := mustParseXML(t, `<root><data Id="mydata">x</data></root>`)
		_, err := xmldsig1.X509DataKeyInfo(cert, nil).BuildKeyInfo(t.Context(), doc, key)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)
	})

	t.Run("SignDetached", func(t *testing.T) {
		doc := mustParseXML(t, `<root><data Id="mydata">x</data></root>`)
		_, err := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{URI: refURIMyData, DigestAlgorithm: xmldsig1.DigestSHA256}).
			KeyInfo(xmldsig1.X509DataKeyInfo(cert, nil)).
			SignDetached(t.Context(), doc, key)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)
	})

	t.Run("SignEnveloping leaves content unmoved", func(t *testing.T) {
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()
		payload, err := doc.CreateElement("payload")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(payload))

		sig, err := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{URI: refURID1, DigestAlgorithm: xmldsig1.DigestSHA256}).
			KeyInfo(xmldsig1.X509DataKeyInfo(nil)).
			SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)
		require.Nil(t, sig)
		require.Equal(t, root, payload.Parent(), "preflight must reject before moving content")
	})
}

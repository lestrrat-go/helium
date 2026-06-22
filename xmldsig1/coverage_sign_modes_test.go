package xmldsig1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestSignEnvelopingRoundTrip drives signEnveloping (content wrapped in an
// Object element) plus KeyInfo construction, then verifies it.
func TestSignEnvelopingRoundTrip(t *testing.T) {
	key := generateRSAKey(t)
	// The document already contains the element the reference points at; the
	// enveloping Object wraps separate content. signEnveloping resolves the
	// reference against the live document tree.
	doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

	payload := doc.CreateElement("Payload")
	require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		SignatureID("sig-1").
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#d1",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			ID:              "ref-1",
			Type:            xmldsig1.TypeObject,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

	sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
	require.NoError(t, err)
	require.NotNil(t, sigElem)

	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// The Signature element carries the configured Id.
	id, ok := sigElem.GetAttribute("Id")
	require.True(t, ok)
	require.Equal(t, "sig-1", id)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestSignDetachedWithKeyInfoAndID drives the full signDetached path including
// the KeyInfo builder branch and Id/Type attributes.
func TestSignDetachedWithKeyInfoAndID(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<root><data Id="mydata">Hello</data></root>`)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		SignatureID("detached-sig").
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#mydata",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			ID:              "r1",
			Type:            xmldsig1.TypeObject,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		KeyInfo(xmldsig1.X509DataKeyInfo())

	sigElem, err := signer.SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestSignWithExclusiveC14NPrefixes drives the InclusiveNamespaces/PrefixList
// branch of processReference and the prefix-roundtrip on verify.
func TestSignWithExclusiveC14NPrefixes(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<root xmlns:a="urn:a" xmlns:b="urn:b"><data Id="d1"><a:x b:attr="v">hi</a:x></data></root>`)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#d1",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform("a", "b")},
		})

	sigElem, err := signer.SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestSignWithC14N11 drives resolveC14NMode's C14N11 arm for both the
// reference transform and the SignedInfo canonicalization method.
func TestSignWithC14N11(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		CanonicalizationMethod(xmldsig1.C14N11URI).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.C14NTransform(xmldsig1.C14N11URI)},
		})
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err := verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestSignWithC14N10 drives resolveC14NMode's plain C14N10 arm.
func TestSignWithC14N10(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		CanonicalizationMethod(xmldsig1.C14N10).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.C14NTransform(xmldsig1.C14N10)},
		})
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err := verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestSignReferenceNotFound drives processReference's resolveReference error
// path (a fragment URI matching no element).
func TestSignReferenceNotFound(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#does-not-exist",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})
	_, err := signer.SignDetached(t.Context(), doc, key)
	require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)
}

// TestSignExternalReferenceRejected drives resolveReference's external-URI
// (non-fragment) rejection branch.
func TestSignExternalReferenceRejected(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "http://example.com/external.xml",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})
	_, err := signer.SignDetached(t.Context(), doc, key)
	require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)
}

// unsupportedTransform is a Transform whose URI is one the verifier does not
// recognize. The signer treats an unrecognized Transform as the default
// Exclusive C14N for canonicalization but still writes the URI into the
// document, so the resulting (self-consistent) signature triggers
// verifyReference's fail-closed unsupported-transform rejection.
type unsupportedTransform struct{}

func (unsupportedTransform) URI() string { return xmldsig1.TransformXPath }

// TestVerifyUnsupportedTransform drives verifyReference's fail-closed
// unsupported-transform branch.
func TestVerifyUnsupportedTransform(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), unsupportedTransform{}},
		})
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err := verifier.Verify(t.Context(), doc)
	require.ErrorIs(t, err, xmldsig1.ErrUnsupportedTransform)
}

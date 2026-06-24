package xmldsig1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	// enveloping drives signEnveloping (content wrapped in an Object element)
	// plus KeyInfo construction, then verifies it.
	t.Run("enveloping", func(t *testing.T) {
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
	})

	// detached with KeyInfo and ID drives the full signDetached path including
	// the KeyInfo builder branch and Id/Type attributes.
	t.Run("detached with keyinfo and id", func(t *testing.T) {
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
	})

	// exclusive c14n prefixes drives the InclusiveNamespaces/PrefixList branch of
	// processReference and the prefix-roundtrip on verify.
	t.Run("exclusive c14n prefixes", func(t *testing.T) {
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
	})

	// c14n11 drives resolveC14NMode's C14N11 arm for both the reference transform
	// and the SignedInfo canonicalization method.
	t.Run("c14n11", func(t *testing.T) {
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
	})

	// c14n10 drives resolveC14NMode's plain C14N10 arm.
	t.Run("c14n10", func(t *testing.T) {
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
	})

	// reference not found drives processReference's resolveReference error path (a
	// fragment URI matching no element).
	t.Run("reference not found", func(t *testing.T) {
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
	})

	// external reference rejected drives resolveReference's external-URI
	// (non-fragment) rejection branch.
	t.Run("external reference rejected", func(t *testing.T) {
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
	})
}

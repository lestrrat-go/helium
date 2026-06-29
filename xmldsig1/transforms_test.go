package xmldsig1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestSignVerifyNamespacedSubtree exercises a fragment reference into a
// namespace-qualified subtree across canonicalization modes.
func TestSignVerifyNamespacedSubtree(t *testing.T) {
	// exclusive c14n: the canonicalized subtree must retain its xmlns
	// declarations so the digest matches a W3C-conformant canonicalization.
	t.Run("exclusive c14n", func(t *testing.T) {
		const xml = `<doc xmlns:p="urn:p"><p:target Id="x"><p:child>v</p:child></p:target></doc>`
		key := generateRSAKey(t)
		doc := mustParseXML(t, xml)

		ref := xmldsig1.ReferenceConfig{
			URI:             "#x",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(ref)

		sigElem, err := signer.SignDetached(t.Context(), doc, key)
		require.NoError(t, err)
		require.NotNil(t, sigElem)

		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// inclusive c14n covers the inclusive C14N 1.0 path for a namespaced subtree.
	t.Run("inclusive c14n", func(t *testing.T) {
		const xml = `<doc xmlns:p="urn:p"><p:target Id="x"><p:child>v</p:child></p:target></doc>`
		key := generateRSAKey(t)
		doc := mustParseXML(t, xml)

		ref := xmldsig1.ReferenceConfig{
			URI:             "#x",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.C14NTransform(xmldsig1.C14N10)},
		}

		signer := xmldsig1.NewSigner().
			CanonicalizationMethod(xmldsig1.C14N10).
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(ref)

		sigElem, err := signer.SignDetached(t.Context(), doc, key)
		require.NoError(t, err)

		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

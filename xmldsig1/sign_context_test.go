package xmldsig1_test

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestSignHonorsContextCancellation ensures every signing entry point honors an
// already-cancelled context the same way verification does: it short-circuits at
// the top of the call, before building the Signature skeleton, moving any caller
// content, canonicalizing, or running the crypto sign, and surfaces the context
// error. Each mode also proves the caller's DOM is left untouched — an enveloped
// sign adds no Signature, an enveloping sign moves no content into an <Object>.
// The per-Reference loop repeats the same check between references (see the
// multi-reference subtest), mirroring verifySignature's per-Reference guard.
func TestSignHonorsContextCancellation(t *testing.T) {
	t.Run("SignEnveloped", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		parent := doc.DocumentElement()

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())

		// Sanity: under a live context the sign succeeds and adds a Signature.
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, parent, key))
		require.NotNil(t, findSignatureElement(parent), "live sign must add a Signature")

		// A cancelled context aborts before any work; the parent gains no further
		// Signature and the error is the context error.
		doc2 := mustParseXML(t, samlAssertion)
		parent2 := doc2.DocumentElement()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := signer.SignEnveloped(ctx, doc2, parent2, key)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, findSignatureElement(parent2), "cancelled sign must not add a Signature")
	})

	t.Run("SignEnveloping", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()

		payload, err := doc.CreateElement("payload")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(payload))

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			})

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		sig, err := signer.SignEnveloping(ctx, doc, []helium.Node{payload}, key)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, sig)
		// The content node was never moved into an <Object>; it stays under root.
		require.Equal(t, root, payload.Parent(), "cancelled sign must not move caller content")
	})

	t.Run("SignDetached", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="mydata">Hello</data></root>`)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURIMyData,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			})

		// Sanity: under a live context the sign succeeds.
		sig, err := signer.SignDetached(t.Context(), doc, key)
		require.NoError(t, err)
		require.NotNil(t, sig)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		sig, err = signer.SignDetached(ctx, doc, key)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, sig)
	})

	// The per-Reference loop checks ctx between references: a detached signature
	// over several references must not be canonicalized and digested to completion
	// once the caller's context is cancelled.
	t.Run("multiple references", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><a Id="one">first</a><b Id="two">second</b></root>`)

		mkRef := func(uri string) xmldsig1.ReferenceConfig {
			return xmldsig1.ReferenceConfig{
				URI:             uri,
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}
		}

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(mkRef("#one")).
			Reference(mkRef("#two"))

		// Sanity: verifies under a live context.
		_, err := signer.SignDetached(t.Context(), doc, key)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = signer.SignDetached(ctx, doc, key)
		require.ErrorIs(t, err, context.Canceled)
	})
}

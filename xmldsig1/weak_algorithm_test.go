package xmldsig1_test

import (
	"context"
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// countLocalElements reports how many elements (by local name) appear under n.
// Used to assert that a rejected sign call did not graft any signature structure
// onto the caller's nodes.
func countLocalElements(n helium.Node, localName string) int {
	count := 0
	elem, ok := helium.AsNode[*helium.Element](n)
	if !ok {
		return 0
	}
	name := elem.Name()
	for i := range len(name) {
		if name[i] == ':' {
			name = name[i+1:]
			break
		}
	}
	if name == localName {
		count++
	}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		count += countLocalElements(child, localName)
	}
	return count
}

// panicKeySource records whether ResolveKey was invoked and panics if so, so a
// verify path that resolves the key before rejecting weak algorithms is caught.
type panicKeySource struct {
	called *bool
}

func (p panicKeySource) ResolveKey(_ context.Context, _ *xmldsig1.KeyInfoData, _ string) (any, error) {
	*p.called = true
	panic("ResolveKey must not be called for a rejected weak-algorithm signature")
}

// TestWeakAlgorithmPreflight covers the up-front weak-algorithm rejection on the
// sign and verify paths.
func TestWeakAlgorithmPreflight(t *testing.T) {
	// enveloping sha1 leaves input unmutated asserts that a default (SHA-1)
	// SignEnveloping call returns ErrWeakAlgorithm AND leaves the caller's content
	// nodes unmoved — not grafted into a new <Object>, and still children of their
	// original parent.
	t.Run("enveloping sha1 leaves input unmutated", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><payload>hello</payload></root>`)

		root := doc.DocumentElement()
		payload := root.FirstChild()
		require.NotNil(t, payload)
		originalParent := payload.Parent()

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA1).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: xmldsig1.DigestSHA1,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			})

		sig, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)
		require.Nil(t, sig)

		// The payload must still be parented where it was; it must NOT have been
		// moved into an Object element.
		require.Same(t, originalParent, payload.Parent(), "content node was moved despite rejection")
		require.Equal(t, 0, countLocalElements(root, "Object"), "an Object element was created despite rejection")
		require.Equal(t, 0, countLocalElements(root, "Signature"), "a Signature element was created despite rejection")
	})

	// enveloped sha1 digest leaves input unmutated asserts that a SHA-1 digest
	// (with a strong signature algorithm) is also rejected up front for the
	// enveloped path, without adding a Signature element to the parent.
	t.Run("enveloped sha1 digest leaves input unmutated", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		parent := doc.DocumentElement()

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: xmldsig1.DigestSHA1,
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})

		err := signer.SignEnveloped(t.Context(), doc, parent, key)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)
		require.Equal(t, 0, countLocalElements(parent, "Signature"), "a Signature element was added despite rejection")
	})

	// weak digest error carries index and uri asserts that a per-reference
	// weak-digest rejection in the signing preflight identifies WHICH reference
	// failed. Here the FIRST reference uses SHA-256 but the SECOND ("#second")
	// uses SHA-1, so the returned error must both match ErrWeakAlgorithm and
	// expose the failing reference's index (1) and URI through a *ReferenceError.
	t.Run("weak digest error carries index and uri", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">first</data><more Id="second">second</more></root>`)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "#second",
				DigestAlgorithm: xmldsig1.DigestSHA1,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			})
		_, err := signer.SignDetached(t.Context(), doc, key)
		require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)

		var refErr *xmldsig1.ReferenceError
		require.True(t, errors.As(err, &refErr))
		require.Equal(t, 1, refErr.Reference)
		require.Equal(t, "#second", refErr.URI)
	})

	// verify weak digest error carries index and uri asserts that a per-reference
	// weak-digest rejection in the VERIFY preflight identifies WHICH reference
	// failed, symmetric with the sign side. The signature uses a strong signature
	// algorithm (RSA-SHA256) but a SHA-1 digest, so the digest preflight — not the
	// signature-algorithm check — is what rejects it, and the returned error must
	// both match ErrWeakAlgorithm and expose the failing reference's index (0) and
	// URI ("") through a *VerificationError.
	t.Run("verify weak digest error carries index and uri", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			AllowSHA1(true).
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: xmldsig1.DigestSHA1,
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err := verifier.Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)

		var verErr *xmldsig1.VerificationError
		require.True(t, errors.As(err, &verErr))
		require.Equal(t, 0, verErr.Reference)
		require.Equal(t, "", verErr.URI)
	})

	// verify sha1 rejected before key resolution asserts that verifying a SHA-1
	// signature returns ErrWeakAlgorithm WITHOUT invoking KeySource.
	t.Run("verify sha1 rejected before key resolution", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := signRSASHA1Doc(t, key)

		called := false
		verifier := xmldsig1.NewVerifier(panicKeySource{called: &called})

		_, err := verifier.Verify(t.Context(), doc)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrWeakAlgorithm)
		require.False(t, called, "KeySource.ResolveKey was invoked despite weak-algorithm rejection")
	})

	// verify sha1 accepted with opt-in confirms the opt-in path still resolves the
	// key and verifies successfully.
	t.Run("verify sha1 accepted with opt-in", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := signRSASHA1Doc(t, key)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).AllowSHA1(true)
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// enveloping sha1 accepted with opt-in confirms opt-in enveloping signing
	// still works end-to-end.
	t.Run("enveloping sha1 accepted with opt-in", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<wrapper/>`)

		payload, err := doc.CreateElement("payload")
		require.NoError(t, err)
		require.NoError(t, payload.AddChild(doc.CreateText([]byte("data"))))

		signer := xmldsig1.NewSigner().
			AllowSHA1(true).
			SignatureAlgorithm(xmldsig1.AlgRSASHA1).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: xmldsig1.DigestSHA1,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			})

		sig, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.NoError(t, err)
		require.NotNil(t, sig)
		require.Equal(t, 1, countLocalElements(sig, "Object"))
	})
}

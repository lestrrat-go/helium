package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestForeignNamespaceReferenceRejected guards against a namespace-confusion
// bypass of the empty-reference check (see TestEmptyReferencesRejected). The
// "at least one Reference" rule must count only ds:Reference elements in the
// XML-Signature namespace. If parseSignedInfo matched on local name alone, a
// SignedInfo carrying zero genuine ds:Reference children plus a single
// foreign-namespace <evil:Reference> would satisfy len(references) > 0 and
// verify against a recomputed SignatureValue while covering no document
// content — re-opening the no-content-signature bypass.
//
// As with the empty-reference attack, full key control is assumed (the worst
// case): produce a genuine signature, rewrite its only ds:Reference into a
// foreign namespace, then recompute a valid SignatureValue over the mutated
// SignedInfo.
func TestForeignNamespaceReferenceRejected(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             "#payload",
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	signedInfo := findChild(t, sigElem, "SignedInfo")
	ref := findChild(t, signedInfo, "Reference")

	// Move the genuine ds:Reference into a foreign namespace so that, by local
	// name alone, the SignedInfo still appears to contain a Reference while it
	// no longer carries any ds:Reference.
	const evilNS = "urn:example:evil"
	require.NoError(t, ref.DeclareNamespace("evil", evilNS))
	require.NoError(t, ref.SetActiveNamespace("evil", evilNS))
	require.Equal(t, evilNS, elementNamespaceURI(ref))

	// Recompute a valid SignatureValue over the mutated SignedInfo so the only
	// thing standing between this document and a false "verified" result is the
	// namespace check on the Reference count.
	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical)
	require.NoError(t, err)

	sigValueElem := findChild(t, sigElem, "SignatureValue")
	for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
		mc, ok := c.(helium.MutableNode)
		require.True(t, ok)
		helium.UnlinkNode(mc)
	}
	require.NoError(t, sigValueElem.AddChild(
		doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(context.Background(), doc)
	require.Error(t, err, "signature whose only Reference is in a foreign namespace must be rejected")
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.True(t, strings.Contains(err.Error(), "Reference"),
		"error should mention the missing Reference: %v", err)
}

// TestDSig11NamespaceReferenceRejected guards the core/1.1 namespace split. The
// XML-Signature 1.1 namespace (http://www.w3.org/2009/xmldsig11#) is only for
// new 1.1-specific elements (e.g. ECKeyValue), NOT an alternate spelling of the
// core Reference. A dsig11:Reference therefore must not count toward the
// at-least-one-Reference rule, so mutating the only ds:Reference into the 1.1
// namespace must make verification fail just like a foreign-namespace Reference.
func TestDSig11NamespaceReferenceRejected(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             "#payload",
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	signedInfo := findChild(t, sigElem, "SignedInfo")
	ref := findChild(t, signedInfo, "Reference")

	// Move the genuine ds:Reference into the XML-Signature 1.1 namespace. By
	// local name alone the SignedInfo still appears to contain a Reference, but
	// xmldsig11# is not the core namespace, so it must not be accepted as one.
	require.NoError(t, ref.DeclareNamespace("dsig11", NamespaceDSig11))
	require.NoError(t, ref.SetActiveNamespace("dsig11", NamespaceDSig11))
	require.Equal(t, NamespaceDSig11, elementNamespaceURI(ref))

	// Recompute a valid SignatureValue over the mutated SignedInfo so the only
	// thing standing between this document and a false "verified" result is the
	// core-namespace check on the Reference count.
	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical)
	require.NoError(t, err)

	sigValueElem := findChild(t, sigElem, "SignatureValue")
	for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
		mc, ok := c.(helium.MutableNode)
		require.True(t, ok)
		helium.UnlinkNode(mc)
	}
	require.NoError(t, sigValueElem.AddChild(
		doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(context.Background(), doc)
	require.Error(t, err, "signature whose only Reference is in the xmldsig11# namespace must be rejected")
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.True(t, strings.Contains(err.Error(), "Reference"),
		"error should mention the missing Reference: %v", err)
}

// TestGenuineReferenceStillVerifies is the positive control: a normal
// ds:Reference signature must continue to verify after the namespace guard is
// added, so the fix rejects only foreign-namespace References.
func TestGenuineReferenceStillVerifies(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             "#payload",
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(context.Background(), doc)
	require.NoError(t, err, "a genuine ds:Reference signature must still verify")
}

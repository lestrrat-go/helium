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

// TestEmptyReferencesRejected guards against a SignedInfo that carries zero
// Reference children. XML-Signature requires at least one Reference; a
// SignatureValue computed over a reference-free SignedInfo cryptographically
// verifies yet covers no document content, so the signature attests to
// nothing. Verify must reject such a structure rather than report success.
//
// The attack is constructed with full key control (an attacker controlling the
// signing key is the worst case): produce a genuine signature, strip the
// Reference element, then recompute a valid SignatureValue over the now-empty
// SignedInfo. The resulting document is a perfectly valid empty-reference
// signature.
func TestEmptyReferencesRejected(t *testing.T) {
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

	// Locate SignedInfo and remove its single Reference child so the
	// SignedInfo now has zero references.
	signedInfo := findChild(t, sigElem, "SignedInfo")
	ref := findChild(t, signedInfo, "Reference")
	helium.UnlinkNode(ref)

	// Recompute a valid SignatureValue over the reference-free SignedInfo.
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

	// Sanity: the signature value itself is cryptographically valid over the
	// empty SignedInfo, so the only thing standing between this document and a
	// false "verified" result is the empty-reference check.
	require.Equal(t, 0, countChildren(signedInfo, "Reference"))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(context.Background(), doc)
	require.Error(t, err, "signature with zero references must be rejected")
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.True(t, strings.Contains(err.Error(), "Reference"),
		"error should mention the missing Reference: %v", err)
}

func findChild(t *testing.T, parent *helium.Element, name string) *helium.Element {
	t.Helper()
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if localName(e) == name {
			return e
		}
	}
	t.Fatalf("child %q not found", name)
	return nil
}

func countChildren(parent *helium.Element, name string) int {
	n := 0
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if localName(e) == name {
			n++
		}
	}
	return n
}

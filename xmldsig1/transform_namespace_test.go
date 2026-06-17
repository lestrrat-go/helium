package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// findTransformByAlgorithm locates the ds:Transform element inside a Reference
// whose Algorithm attribute matches algURI, so a test can mutate that specific
// transform.
func findTransformByAlgorithm(t *testing.T, signedInfo *helium.Element, algURI string) *helium.Element {
	t.Helper()
	ref := findChild(t, signedInfo, "Reference")
	transforms := findChild(t, ref, "Transforms")
	for c := transforms.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if localName(e) != "Transform" {
			continue
		}
		alg, _ := e.GetAttribute("Algorithm")
		if alg == algURI {
			return e
		}
	}
	t.Fatalf("Transform with Algorithm %q not found", algURI)
	return nil
}

// TestForeignNamespaceTransformRejected guards against a namespace-confusion
// bypass where an attacker rewrites a core ds:Transform into a foreign
// namespace. A Transform element is itself in the XML-Signature namespace, so
// parseReferenceElement must honor only ds:Transform elements; matching on
// local name alone would let an <evil:Transform Algorithm="...enveloped...">
// drive a privileged transform.
//
// The attack: an enveloped signature relies on the enveloped-signature
// Transform to detach the Signature element before digesting. If a foreign
// <evil:Transform> carrying the enveloped-signature algorithm is honored, the
// Signature is detached and the recomputed digest matches. Once the guard
// ignores the foreign transform, the Signature element is no longer detached,
// so the canonical bytes include it and the digest no longer matches — the
// forgery is rejected. Full key control is assumed (worst case): recompute a
// valid SignatureValue over the mutated SignedInfo.
func TestForeignNamespaceTransformRejected(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data>secret</data></root>`))
	require.NoError(t, err)

	signer := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             "",
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{Enveloped(), ExcC14NTransform()},
		})
	require.NoError(t, signer.SignEnveloped(context.Background(), doc, doc.DocumentElement(), key))

	sigElem := findChild(t, doc.DocumentElement(), "Signature")
	signedInfo := findChild(t, sigElem, "SignedInfo")

	// Move the enveloped-signature Transform into a foreign namespace so that,
	// by local name alone, it still looks like a Transform while it is no
	// longer a genuine ds:Transform.
	envTransform := findTransformByAlgorithm(t, signedInfo, TransformEnvelopedSignature)
	const evilNS = "urn:example:evil"
	require.NoError(t, envTransform.DeclareNamespace("evil", evilNS))
	require.NoError(t, envTransform.SetActiveNamespace("evil", evilNS))
	require.Equal(t, evilNS, elementNamespaceURI(envTransform))

	// Recompute a valid SignatureValue over the mutated SignedInfo so the only
	// thing standing between this document and a false "verified" result is the
	// namespace check on the Transform element.
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
	require.Error(t, err, "signature whose enveloped Transform is in a foreign namespace must be rejected")
	require.ErrorIs(t, err, ErrDigestMismatch)
}

// TestGenuineTransformsStillVerify is the positive control for the Transform
// namespace guard. A genuine enveloped + Exclusive C14N signature, including an
// InclusiveNamespaces child (which lives in the xml-exc-c14n namespace, NOT the
// XML-Signature namespace), must continue to verify — proving the guard rejects
// only foreign-namespace Transform elements and does not reach the exc-c14n
// InclusiveNamespaces child.
func TestGenuineTransformsStillVerify(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(),
		[]byte(`<root xmlns:p="urn:example:p"><p:data>secret</p:data></root>`))
	require.NoError(t, err)

	signer := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             "",
			DigestAlgorithm: DigestSHA256,
			// ExcC14NTransform with a prefix list emits an InclusiveNamespaces
			// child in the xml-exc-c14n namespace, exercising the exc-c14n
			// child path the guard must leave untouched.
			Transforms: []Transform{Enveloped(), ExcC14NTransform("p")},
		})
	require.NoError(t, signer.SignEnveloped(context.Background(), doc, doc.DocumentElement(), key))

	// Confirm the InclusiveNamespaces child really is in the exc-c14n namespace,
	// not the DSig namespace, so the positive control is meaningful.
	sigElem := findChild(t, doc.DocumentElement(), "Signature")
	signedInfo := findChild(t, sigElem, "SignedInfo")
	excTransform := findTransformByAlgorithm(t, signedInfo, ExcC14N10)
	incNS := findChild(t, excTransform, "InclusiveNamespaces")
	require.Equal(t, "http://www.w3.org/2001/10/xml-exc-c14n#", elementNamespaceURI(incNS))
	require.False(t, isDSigCoreNS(incNS), "InclusiveNamespaces must not be in the DSig namespace")

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(context.Background(), doc)
	require.NoError(t, err, "a genuine enveloped + exc-c14n signature with InclusiveNamespaces must still verify")
}

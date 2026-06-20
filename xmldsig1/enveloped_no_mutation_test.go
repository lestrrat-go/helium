package xmldsig1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// findSignatureElement walks the tree and returns the first ds:Signature
// element, or nil if none is present.
func findSignatureElement(root helium.Node) *helium.Element {
	if root == nil {
		return nil
	}
	if e, ok := helium.AsNode[*helium.Element](root); ok {
		if e.LocalName() == "Signature" {
			return e
		}
	}
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if found := findSignatureElement(c); found != nil {
			return found
		}
	}
	return nil
}

func findElementByLocalName(root helium.Node, name string) *helium.Element {
	if root == nil {
		return nil
	}
	if e, ok := helium.AsNode[*helium.Element](root); ok {
		if e.LocalName() == name {
			return e
		}
	}
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if found := findElementByLocalName(c, name); found != nil {
			return found
		}
	}
	return nil
}

// TestVerifyEnvelopedDoesNotMutateDOM is the regression guard for D-SIG-001:
// the enveloped-signature transform must NOT mutate the caller's live document.
// The previous implementation unlinked the Signature element from the live tree
// during canonicalization and reattached it afterward, which races with
// concurrent readers and corrupts the document if the restore ever fails. After
// Verify, the document must be byte-for-byte identical to before, and the
// Signature element must remain linked at its original position with the same
// node identity.
func TestVerifyEnvelopedDoesNotMutateDOM(t *testing.T) {
	key := generateRSAKey(t)

	signDoc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), signDoc, signDoc.DocumentElement(), key))

	signed, err := helium.WriteString(signDoc)
	require.NoError(t, err)

	// Re-parse from the serialized form so the verifier operates on a fresh,
	// independently-owned tree.
	doc := mustParseXML(t, signed)
	before, err := helium.WriteString(doc)
	require.NoError(t, err)

	sigElem := findSignatureElement(doc.DocumentElement())
	require.NotNil(t, sigElem, "Signature must be present before verify")
	sigParentBefore := sigElem.Parent()

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)

	after, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Equal(t, before, after, "Verify must not mutate the caller's DOM")

	sigAfter := findSignatureElement(doc.DocumentElement())
	require.NotNil(t, sigAfter, "Signature must still be linked after verify")
	require.Same(t, sigElem, sigAfter, "Signature element identity must be preserved")
	require.Same(t, sigParentBefore, sigAfter.Parent(), "Signature must remain at its original parent")
}

// TestVerifyEnvelopedIdempotent confirms that, because Verify no longer mutates
// the DOM, repeated verifications of the same document all succeed.
func TestVerifyEnvelopedIdempotent(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	for i := range 3 {
		_, err := verifier.Verify(t.Context(), doc)
		require.NoErrorf(t, err, "verify #%d", i)
	}
}

// TestVerifyEnvelopedFragmentDoesNotMutateDOM exercises the URI="#id" enveloped
// path: the Signature is nested inside the signed element and must be omitted
// from the subtree canonicalization without mutating the live tree.
func TestVerifyEnvelopedFragmentDoesNotMutateDOM(t *testing.T) {
	key := generateRSAKey(t)

	signDoc := mustParseXML(t, `<root><data Id="mydata"><v>hello</v></data></root>`)
	dataElem := findElementByLocalName(signDoc.DocumentElement(), "data")
	require.NotNil(t, dataElem)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#mydata",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
		})
	// Place the Signature inside the referenced element so it is enveloped.
	require.NoError(t, signer.SignEnveloped(t.Context(), signDoc, dataElem, key))

	signed, err := helium.WriteString(signDoc)
	require.NoError(t, err)

	doc := mustParseXML(t, signed)
	before, err := helium.WriteString(doc)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)

	after, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Equal(t, before, after, "Verify must not mutate the caller's DOM for fragment references")
}

// TestVerifyEnvelopedFragmentSignatureBeforeTarget is the regression guard for
// D-SIG-002: when the ds:Signature is a SIBLING that precedes the referenced
// target element, unlinking the cloned Signature shifts the target's child
// index. The enveloped c14n must resolve the cloned target by its pre-unlink
// path (before detaching the Signature), or it resolves the wrong/nil subtree
// and verification fails with "reference target in canonicalization copy is not
// an element". The live DOM must also remain unmutated.
func TestVerifyEnvelopedFragmentSignatureBeforeTarget(t *testing.T) {
	key := generateRSAKey(t)

	signDoc := mustParseXML(t, `<root><x Id="x"><v>hello</v></x></root>`)
	root := signDoc.DocumentElement()
	xElem := findElementByLocalName(root, "x")
	require.NotNil(t, xElem)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#x",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
		})
	// Place the Signature as a child of root; it is appended AFTER <x>.
	require.NoError(t, signer.SignEnveloped(t.Context(), signDoc, root, key))

	// Reorder so the Signature precedes <x> as an earlier sibling. Moving <x>
	// past the Signature does not change <x>'s content or the enveloped omit
	// set, so the digest stays valid — but it forces the unlink-shifts-index
	// condition the fix guards against.
	helium.UnlinkNode(xElem)
	require.NoError(t, root.AddChild(xElem))

	sigElem := findSignatureElement(root)
	require.NotNil(t, sigElem)
	require.Same(t, sigElem, root.FirstChild(), "Signature must now be the first child")

	signed, err := helium.WriteString(signDoc)
	require.NoError(t, err)

	doc := mustParseXML(t, signed)
	before, err := helium.WriteString(doc)
	require.NoError(t, err)

	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err, "verify must resolve the shifted-index target")

	after, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Equal(t, before, after, "Verify must not mutate the caller's DOM")
}

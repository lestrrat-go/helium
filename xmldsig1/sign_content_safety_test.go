package xmldsig1_test

import (
	"context"
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// failingKeyInfo is a KeyInfoBuilder that always fails. It injects an error at
// the LAST signing step (KeyInfo construction), after SignEnveloping has already
// moved the caller's content into the <ds:Object>, so it exercises the
// restore-on-error path.
type failingKeyInfo struct {
	err error
}

func (f failingKeyInfo) BuildKeyInfo(context.Context, *helium.Document, any) (*helium.Element, error) {
	return nil, f.err
}

// TestSignEnvelopingContentSafety covers the data-integrity guarantees of
// SignEnveloping: the caller's content is restored to its exact original tree
// position when signing fails after the content has been moved, and a nil or
// non-movable content entry is rejected up front instead of being silently
// dropped.
func TestSignEnvelopingContentSafety(t *testing.T) {
	// A failure AFTER the content is moved into the <ds:Object> must leave the
	// caller's DOM byte-identical to before the call: every moved node back under
	// its original parent, between its original siblings, in the original order.
	t.Run("restores content on error", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()

		before, err := doc.CreateElement("before")
		require.NoError(t, err)
		payloadOne, err := doc.CreateElement("payloadOne")
		require.NoError(t, err)
		require.NoError(t, payloadOne.AddChild(doc.CreateText([]byte("one"))))
		payloadTwo, err := doc.CreateElement("payloadTwo")
		require.NoError(t, err)
		require.NoError(t, payloadTwo.AddChild(doc.CreateText([]byte("two"))))
		after, err := doc.CreateElement("after")
		require.NoError(t, err)

		// Child order under root after these appends: data, before, payloadOne,
		// payloadTwo, after.
		require.NoError(t, root.AddChild(before))
		require.NoError(t, root.AddChild(payloadOne))
		require.NoError(t, root.AddChild(payloadTwo))
		require.NoError(t, root.AddChild(after))
		originalChildren := childElements(root)

		wantErr := errors.New("keyinfo boom")
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			}).
			KeyInfo(failingKeyInfo{err: wantErr})

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payloadOne, payloadTwo}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		// Both payloads are back under root in their original positions.
		require.Equal(t, root, payloadOne.Parent())
		require.Equal(t, root, payloadTwo.Parent())
		require.Equal(t, before, payloadOne.PrevSibling())
		require.Equal(t, payloadTwo, payloadOne.NextSibling())
		require.Equal(t, payloadOne, payloadTwo.PrevSibling())
		require.Equal(t, after, payloadTwo.NextSibling())

		// The surrounding siblings point back at the restored payloads.
		require.Equal(t, payloadOne, before.NextSibling())
		require.Equal(t, payloadTwo, after.PrevSibling())

		// The whole child list of root is back in its original order.
		require.Equal(t, originalChildren, childElements(root))
	})

	// A nil content entry is rejected with an indexed error before any node is
	// moved; signing must NOT silently succeed with content missing from the
	// Object, and any valid sibling entry must stay untouched in the caller tree.
	t.Run("rejects nil content entry", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()

		valid, err := doc.CreateElement("valid")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(valid))

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			})

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{valid, nil}, key)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
		require.Contains(t, err.Error(), "content[1]")
		require.Nil(t, sigElem)

		// The valid entry was never moved: preflight rejected before any move.
		require.Equal(t, root, valid.Parent())
	})

	// A non-movable (read-only) content entry is rejected the same way. A
	// namespace-node wrapper implements helium.Node but NOT helium.MutableNode,
	// so it cannot be relinked into the Object.
	t.Run("rejects non-movable content entry", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

		ns := helium.NewNamespace("x", "urn:x")
		readOnly := helium.NewNamespaceNodeWrapper(ns, doc.DocumentElement())

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			})

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{readOnly}, key)
		require.Error(t, err)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidSignature)
		require.Contains(t, err.Error(), "content[0]")
		require.Nil(t, sigElem)
	})

	// Regression: a successful SignEnveloping still MOVES the content into the
	// returned Signature's <Object> (enveloping semantics) and produces a
	// signature that verifies.
	t.Run("success moves content into object", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

		payload, err := doc.CreateElement("Payload")
		require.NoError(t, err)
		require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			}).
			KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.NoError(t, err)
		require.NotNil(t, sigElem)

		// payload now lives under the Object, which is a child of the Signature.
		objElem, ok := payload.Parent().(*helium.Element)
		require.True(t, ok)
		require.Equal(t, "Object", objElem.LocalName())
		require.Equal(t, sigElem, objElem.Parent())

		require.NoError(t, doc.DocumentElement().AddChild(sigElem))
		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})
}

// childElements returns the ordered child nodes of n.
func childElements(n helium.Node) []helium.Node {
	var out []helium.Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, c)
	}
	return out
}

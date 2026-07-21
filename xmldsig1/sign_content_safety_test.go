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

// TestSignEnvelopingRollbackFidelity covers rollback cases where a naive
// move-and-restore corrupts the caller's tree: adjacent Text entries coalescing
// during the move or the restore, content passed out of document order, and a
// read-only next sibling as the restore anchor. Every case injects a KeyInfo
// error after the content has been moved into the <ds:Object>, then asserts the
// caller's DOM is byte-identical to before the call.
func TestSignEnvelopingRollbackFidelity(t *testing.T) {
	newFailingSigner := func(t *testing.T) (xmldsig1.Signer, error) {
		t.Helper()
		wantErr := errors.New("keyinfo boom")
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			}).
			KeyInfo(failingKeyInfo{err: wantErr})
		return signer, wantErr
	}

	// Two Text nodes handed in as content coalesce when the second is moved into
	// the <ds:Object> next to the first (AddChild merges adjacent text), so a
	// naive move corrupts the first node's content to "firstsecond" and detaches
	// the second before the failure even happens. The move must be
	// non-coalescing so rollback restores each node's original content.
	t.Run("adjacent text content restored without coalescing", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()

		firstText := doc.CreateText([]byte("first"))
		mid, err := doc.CreateElement("mid")
		require.NoError(t, err)
		secondText := doc.CreateText([]byte("second"))

		// root children: data, firstText, mid, secondText (an element separates the
		// two text nodes so they are not adjacent in the caller tree).
		require.NoError(t, root.AddChild(firstText))
		require.NoError(t, root.AddChild(mid))
		require.NoError(t, root.AddChild(secondText))
		originalChildren := childElements(root)

		signer, wantErr := newFailingSigner(t)
		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{firstText, secondText}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		require.Equal(t, "first", string(firstText.Content()))
		require.Equal(t, "second", string(secondText.Content()))
		require.Equal(t, root, firstText.Parent())
		require.Equal(t, root, secondText.Parent())
		require.Equal(t, mid, firstText.NextSibling())
		require.Equal(t, firstText, mid.PrevSibling())
		require.Equal(t, mid, secondText.PrevSibling())
		require.Nil(t, secondText.NextSibling())
		require.Equal(t, originalChildren, childElements(root))
	})

	// A content node that was the last child must be restored without coalescing
	// into its previous Text sibling. AddChild-based restore would merge the
	// restored Text node into the sibling and leave it detached.
	t.Run("last-child text restored without merging into previous sibling", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()

		firstText := doc.CreateText([]byte("first"))
		secondText := doc.CreateText([]byte("second"))
		require.NoError(t, root.AddChild(firstText))
		// Splice secondText right after firstText WITHOUT coalescing (Replace is a
		// pointer-level insert), producing two adjacent Text children.
		require.NoError(t, firstText.Replace(firstText, secondText))
		originalChildren := childElements(root)

		signer, wantErr := newFailingSigner(t)
		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{secondText}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		require.Equal(t, "first", string(firstText.Content()))
		require.Equal(t, "second", string(secondText.Content()))
		require.Equal(t, root, secondText.Parent())
		require.Equal(t, firstText, secondText.PrevSibling())
		require.Equal(t, secondText, firstText.NextSibling())
		require.Nil(t, secondText.NextSibling())
		require.Equal(t, originalChildren, childElements(root))
	})

	// Content passed in reverse document order must still be restored to its
	// original order. Recording each node's anchor at move time captures a stale
	// sibling link (moving an earlier node changes a later node's siblings), so
	// the positions must be snapshotted before any node is moved.
	t.Run("reverse-order content restored in original order", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()

		first, err := doc.CreateElement("first")
		require.NoError(t, err)
		second, err := doc.CreateElement("second")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(first))
		require.NoError(t, root.AddChild(second))
		originalChildren := childElements(root)

		signer, wantErr := newFailingSigner(t)
		// Reversed relative to document order.
		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{second, first}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		require.Equal(t, root, first.Parent())
		require.Equal(t, root, second.Parent())
		require.Equal(t, second, first.NextSibling())
		require.Equal(t, first, second.PrevSibling())
		require.Equal(t, originalChildren, childElements(root))
	})

	// Two adjacent content nodes with no sibling before them (they open the
	// parent's child list) must still be restored in order. The leftmost node has
	// no previous sibling to anchor on, so the restore must resolve the pair from
	// the right without deadlocking.
	t.Run("adjacent leading content restored in order", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<data Id="d1"><first/><second/></data>`)
		root := doc.DocumentElement()

		first, ok := root.FirstChild().(*helium.Element)
		require.True(t, ok)
		require.Equal(t, "first", first.LocalName())
		second, ok := first.NextSibling().(*helium.Element)
		require.True(t, ok)
		require.Equal(t, "second", second.LocalName())
		originalChildren := childElements(root)

		signer, wantErr := newFailingSigner(t)
		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{first, second}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		require.Equal(t, root, first.Parent())
		require.Equal(t, root, second.Parent())
		require.Nil(t, first.PrevSibling())
		require.Equal(t, second, first.NextSibling())
		require.Equal(t, first, second.PrevSibling())
		require.Equal(t, originalChildren, childElements(root))
	})

	// The original next sibling of a content node may be a read-only node (a
	// namespace-node wrapper) that cannot anchor an insert-before. Restore must
	// still land the node at its exact original position by anchoring on the
	// previous sibling instead of appending it after the read-only node.
	t.Run("restored before a read-only next sibling", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		root := doc.DocumentElement()

		before, err := doc.CreateElement("before")
		require.NoError(t, err)
		payload, err := doc.CreateElement("payload")
		require.NoError(t, err)
		ns := helium.NewNamespace("x", "urn:x")
		readOnly := helium.NewNamespaceNodeWrapper(ns, root)
		require.NoError(t, root.AddChild(before))
		require.NoError(t, root.AddChild(payload))
		// Splice the read-only wrapper in as payload's next sibling.
		require.NoError(t, root.AddChild(readOnly))
		originalChildren := childElements(root)

		signer, wantErr := newFailingSigner(t)
		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		require.Equal(t, root, payload.Parent())
		require.Equal(t, before, payload.PrevSibling())
		require.Equal(t, payload, before.NextSibling())
		require.Equal(t, helium.Node(readOnly), payload.NextSibling())
		require.Equal(t, originalChildren, childElements(root))
	})
}

// TestSignEnvelopingRollbackLinkage covers rollback for ordinary caller content
// whose original linkage is not a plain "child under a parent": a detached but
// sibling-linked chain, and content owned by a different helium.Document. Each
// case injects a KeyInfo error after the content has been moved into the
// <ds:Object>, then asserts the original linkage (sibling anchors, owner
// document) is restored exactly.
func TestSignEnvelopingRollbackLinkage(t *testing.T) {
	newFailingSigner := func(t *testing.T) (xmldsig1.Signer, error) {
		t.Helper()
		wantErr := errors.New("keyinfo boom")
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
			}).
			KeyInfo(failingKeyInfo{err: wantErr})
		return signer, wantErr
	}

	// A parentless node that is nonetheless linked to a sibling (an AddSibling
	// chain never attached to any parent) must be spliced back onto that sibling
	// chain, not just unlinked. Only the first element is signed; after rollback
	// the first<->second sibling linkage must be exactly restored.
	t.Run("detached sibling-linked chain restored", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

		first, err := doc.CreateElement("first")
		require.NoError(t, err)
		second, err := doc.CreateElement("second")
		require.NoError(t, err)
		// Link the two as siblings without ever attaching them to a parent.
		require.NoError(t, first.AddSibling(second))
		require.Nil(t, first.Parent())
		require.Nil(t, second.Parent())
		require.Equal(t, helium.Node(second), first.NextSibling())
		require.Equal(t, helium.Node(first), second.PrevSibling())

		signer, wantErr := newFailingSigner(t)
		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{first}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		// The detached sibling chain is restored to its exact original linkage.
		require.Nil(t, first.Parent())
		require.Nil(t, second.Parent())
		require.Equal(t, helium.Node(second), first.NextSibling())
		require.Equal(t, helium.Node(first), second.PrevSibling())
	})

	// Content attached to a DIFFERENT helium.Document has its owner rewritten to
	// the signing document while canonicalizing the detached subtree. After
	// rollback the content's OwnerDocument must be restored to its original
	// document, not left pointing at the signing document.
	t.Run("cross-document content owner restored", func(t *testing.T) {
		key := generateRSAKey(t)
		signDoc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		otherDoc := mustParseXML(t, `<other><payload>content</payload></other>`)

		payload, ok := otherDoc.DocumentElement().FirstChild().(*helium.Element)
		require.True(t, ok)
		require.Equal(t, "payload", payload.LocalName())
		require.Equal(t, otherDoc, payload.OwnerDocument())
		otherRoot := payload.Parent()

		signer, wantErr := newFailingSigner(t)
		sigElem, err := signer.SignEnveloping(t.Context(), signDoc, []helium.Node{payload}, key)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, sigElem)

		// The payload is back under its original parent in the original document,
		// and its OwnerDocument points at that document, not the signing document.
		require.Equal(t, otherRoot, payload.Parent())
		require.Equal(t, otherDoc, payload.OwnerDocument())
		// A descendant's owner is restored too (SetTreeDoc walks the subtree).
		require.Equal(t, otherDoc, payload.FirstChild().OwnerDocument())
	})

	// When the last content node in a slice falls to the parent-append fallback
	// during restore (its original next sibling is still pending and its prev
	// anchor is not yet back), the append must be non-coalescing: appending a Text
	// node next to a residual last Text child must NOT merge them. Three adjacent
	// Text children A/B/C are built with the public non-coalescing Replace; passing
	// [B,C] (and [C,B]) as content with a failing KeyInfoBuilder must restore all
	// three children in exact order with exact per-node content.
	t.Run("adjacent text fallback restored without coalescing", func(t *testing.T) {
		runAdjacentTextFallback := func(t *testing.T, order func(a, b helium.Node) []helium.Node) {
			t.Helper()
			key := generateRSAKey(t)
			doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
			root := doc.DocumentElement()

			textA := doc.CreateText([]byte("AAA"))
			textB := doc.CreateText([]byte("BBB"))
			textC := doc.CreateText([]byte("CCC"))
			// Build three adjacent Text children WITHOUT coalescing (Replace is a
			// pointer-level insert): A, B, C.
			require.NoError(t, root.AddChild(textA))
			require.NoError(t, textA.Replace(textA, textB))
			require.NoError(t, textB.Replace(textB, textC))
			originalChildren := childElements(root)

			signer, wantErr := newFailingSigner(t)
			sigElem, err := signer.SignEnveloping(t.Context(), doc, order(textB, textC), key)
			require.ErrorIs(t, err, wantErr)
			require.Nil(t, sigElem)

			// All three children back under root, in order, with exact content and
			// none coalesced or detached.
			require.Equal(t, "AAA", string(textA.Content()))
			require.Equal(t, "BBB", string(textB.Content()))
			require.Equal(t, "CCC", string(textC.Content()))
			require.Equal(t, root, textA.Parent())
			require.Equal(t, root, textB.Parent())
			require.Equal(t, root, textC.Parent())
			require.Equal(t, textB, textA.NextSibling())
			require.Equal(t, textA, textB.PrevSibling())
			require.Equal(t, textC, textB.NextSibling())
			require.Equal(t, textB, textC.PrevSibling())
			require.Nil(t, textC.NextSibling())
			require.Equal(t, originalChildren, childElements(root))
		}

		t.Run("content order [B,C]", func(t *testing.T) {
			runAdjacentTextFallback(t, func(a, b helium.Node) []helium.Node {
				return []helium.Node{a, b}
			})
		})
		t.Run("content order [C,B]", func(t *testing.T) {
			runAdjacentTextFallback(t, func(a, b helium.Node) []helium.Node {
				return []helium.Node{b, a}
			})
		})
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

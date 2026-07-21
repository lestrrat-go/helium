package xmldsig1_test

import (
	"context"
	"crypto/elliptic"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// refURID1 is the same-document reference URI ("#d1") used across the signing
// tests, pointing at the <data Id="d1"> element in their fixture documents.
const refURID1 = "#d1"

// parentInspectingKeyInfo is a custom KeyInfoBuilder that records the parent of
// a watched content node at the moment BuildKeyInfo is invoked. It proves the
// timing contract an arbitrary caller builder observes: for an enveloping
// signature BuildKeyInfo must run AFTER the content is wrapped in the <Object>,
// so the watched node's parent is the Object element.
type parentInspectingKeyInfo struct {
	watched    helium.Node
	seenParent helium.Node
}

func (b *parentInspectingKeyInfo) BuildKeyInfo(_ context.Context, doc *helium.Document, _ any) (*helium.Element, error) {
	b.seenParent = b.watched.Parent()
	keyInfo, err := doc.CreateElement("KeyInfo")
	if err != nil {
		return nil, err
	}
	if err := keyInfo.DeclareNamespace("ds", xmldsig1.NamespaceDSig); err != nil {
		return nil, err
	}
	if err := keyInfo.SetActiveNamespace("ds", xmldsig1.NamespaceDSig); err != nil {
		return nil, err
	}
	return keyInfo, nil
}

func TestSign(t *testing.T) {
	// enveloping drives signEnveloping (content wrapped in an Object element)
	// plus KeyInfo construction, then verifies it.
	t.Run("enveloping", func(t *testing.T) {
		key := generateRSAKey(t)
		// The document already contains the element the reference points at; the
		// enveloping Object wraps separate content. signEnveloping resolves the
		// reference against the live document tree.
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

		payload, err := doc.CreateElement("Payload")
		require.NoError(t, err)
		require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			SignatureID("sig-1").
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
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

	// enveloping with KeyInfo must place KeyInfo before the Object element so the
	// Signature child order matches the XML-DSig schema content model
	// (SignedInfo, SignatureValue, KeyInfo?, Object*).
	t.Run("enveloping keyinfo precedes object", func(t *testing.T) {
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
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}).
			KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.NoError(t, err)

		var order []string
		for c := sigElem.FirstChild(); c != nil; c = c.NextSibling() {
			if e, ok := c.(*helium.Element); ok {
				order = append(order, e.LocalName())
			}
		}
		require.Equal(t, []string{"SignedInfo", "SignatureValue", "KeyInfo", "Object"}, order)

		require.NoError(t, doc.DocumentElement().AddChild(sigElem))
		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// enveloping with an empty X509DataKeyInfo (→ ErrInvalidKeyInfo) must leave
	// the caller's content node under its ORIGINAL parent. A narrow preflight
	// detects the empty x509DataKeyInfo before the <Object> is created or any
	// content is moved, so the error strands nothing.
	t.Run("enveloping keyinfo error leaves content unmoved", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

		payload, err := doc.CreateElement("Payload")
		require.NoError(t, err)
		require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))
		// Parent the payload under <root> so the test can confirm the failed sign
		// leaves it exactly where it started.
		root := doc.DocumentElement()
		require.NoError(t, root.AddChild(payload))
		require.Equal(t, helium.Node(root), payload.Parent())

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}).
			KeyInfo(xmldsig1.X509DataKeyInfo())

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)
		require.Nil(t, sigElem)
		// The payload was NOT stranded under a detached <Object>; it stays under
		// its original <root> parent.
		require.Equal(t, helium.Node(root), payload.Parent())
	})

	// An arbitrary caller-provided KeyInfoBuilder must observe the established
	// timing contract: BuildKeyInfo runs AFTER the content is wrapped in the
	// <Object>, so a builder inspecting the content sees its parent as the
	// Object element. The narrow empty-X509DataKeyInfo preflight must NOT move
	// this general builder timing earlier.
	t.Run("enveloping general keyinfo builder sees wrapped content", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

		payload, err := doc.CreateElement("Payload")
		require.NoError(t, err)
		require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

		builder := &parentInspectingKeyInfo{watched: payload}
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}).
			KeyInfo(builder)

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.NoError(t, err)
		require.NotNil(t, sigElem)

		// BuildKeyInfo must have observed the payload already wrapped in <Object>.
		parent, ok := builder.seenParent.(*helium.Element)
		require.True(t, ok)
		require.Equal(t, "Object", parent.LocalName())
	})

	// detached with KeyInfo and ID drives the full signDetached path including
	// the KeyInfo builder branch and Id/Type attributes.
	t.Run("detached with keyinfo and id", func(t *testing.T) {
		key := generateRSAKey(t)
		cert := generateSelfSignedCert(t, key)
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
			KeyInfo(xmldsig1.X509DataKeyInfo(cert))

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
				URI:             refURID1,
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

	// A per-reference signing failure must identify WHICH reference failed
	// (index + URI), symmetric with verification's VerificationError. Here the
	// FIRST reference resolves but the SECOND ("#does-not-exist") does not, so
	// the returned error must both match ErrReferenceNotFound and expose the
	// failing reference's index (1) and URI through a *ReferenceError.
	t.Run("reference error carries index and uri", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			}).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "#does-not-exist",
				DigestAlgorithm: xmldsig1.DigestSHA256,
				Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
			})
		_, err := signer.SignDetached(t.Context(), doc, key)
		require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)

		var refErr *xmldsig1.ReferenceError
		require.ErrorAs(t, err, &refErr)
		require.Equal(t, 1, refErr.Reference)
		require.Equal(t, "#does-not-exist", refErr.URI)
	})

	// invalid transform pipeline on enveloping is preflighted: a transform list
	// rejected by resolveTransformPipeline (here Enveloped ordered after a c14n
	// transform) must return ErrUnsupportedTransform WITHOUT moving the caller's
	// content into an <Object> or otherwise mutating the input document.
	t.Run("invalid transform pipeline leaves caller DOM unchanged", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		payload, err := doc.CreateElement("Payload")
		require.NoError(t, err)
		require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             refURID1,
				DigestAlgorithm: xmldsig1.DigestSHA256,
				// c14n transform produces octets; the trailing Enveloped is
				// ordered after canonicalization and is rejected.
				Transforms: []xmldsig1.Transform{xmldsig1.ExcC14NTransform(), xmldsig1.Enveloped()},
			})

		sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
		require.ErrorIs(t, err, xmldsig1.ErrUnsupportedTransform)
		require.Nil(t, sigElem)

		// The caller's payload was never moved into an <Object>.
		require.Nil(t, payload.Parent())
		// The input document is byte-for-byte unchanged (no Signature added).
		require.Nil(t, findSignatureElement(doc.DocumentElement()))
		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	// caller document is not mutated by a detached or enveloping signature: the
	// returned Signature is never inserted into the caller's document (fu10), on
	// both the success and the signing-error paths.
	t.Run("caller document not mutated", func(t *testing.T) {
		newSigner := func() xmldsig1.Signer {
			return xmldsig1.NewSigner().
				SignatureAlgorithm(xmldsig1.AlgRSASHA256).
				Reference(xmldsig1.ReferenceConfig{
					URI:             refURID1,
					DigestAlgorithm: xmldsig1.DigestSHA256,
					Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
				})
		}

		t.Run("detached success", func(t *testing.T) {
			key := generateRSAKey(t)
			doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
			before, err := helium.WriteString(doc)
			require.NoError(t, err)

			sigElem, err := newSigner().SignDetached(t.Context(), doc, key)
			require.NoError(t, err)
			require.NotNil(t, sigElem)
			require.Nil(t, sigElem.Parent(), "detached Signature must not be linked into the caller document")

			require.Nil(t, findSignatureElement(doc.DocumentElement()), "caller document must carry no Signature")
			after, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, before, after, "caller document must be byte-unchanged after a detached signature")
		})

		t.Run("enveloping success", func(t *testing.T) {
			key := generateRSAKey(t)
			doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
			before, err := helium.WriteString(doc)
			require.NoError(t, err)

			payload, err := doc.CreateElement("Payload")
			require.NoError(t, err)
			require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

			sigElem, err := newSigner().SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
			require.NoError(t, err)
			require.NotNil(t, sigElem)
			require.Nil(t, sigElem.Parent(), "enveloping Signature must not be linked into the caller document")

			require.Nil(t, findSignatureElement(doc.DocumentElement()), "caller document must carry no Signature")
			after, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, before, after, "caller document must be byte-unchanged after an enveloping signature")
		})

		// An RSA signature algorithm with an ECDSA key fails inside SignedInfo
		// signing, after canonicalization. The caller document must stay untouched.
		t.Run("detached signing error", func(t *testing.T) {
			ecKey := generateECDSAKey(t, elliptic.P256())
			doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
			before, err := helium.WriteString(doc)
			require.NoError(t, err)

			sigElem, err := newSigner().SignDetached(t.Context(), doc, ecKey)
			require.ErrorIs(t, err, xmldsig1.ErrKeyMismatch)
			require.Nil(t, sigElem)

			require.Nil(t, findSignatureElement(doc.DocumentElement()), "caller document must carry no Signature")
			after, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, before, after, "caller document must be byte-unchanged after a failed detached signature")
		})

		t.Run("enveloping signing error", func(t *testing.T) {
			ecKey := generateECDSAKey(t, elliptic.P256())
			doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)
			before, err := helium.WriteString(doc)
			require.NoError(t, err)

			payload, err := doc.CreateElement("Payload")
			require.NoError(t, err)
			require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

			sigElem, err := newSigner().SignEnveloping(t.Context(), doc, []helium.Node{payload}, ecKey)
			require.ErrorIs(t, err, xmldsig1.ErrKeyMismatch)
			require.Nil(t, sigElem)

			require.Nil(t, findSignatureElement(doc.DocumentElement()), "caller document must carry no Signature")
			after, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, before, after, "caller document must be byte-unchanged after a failed enveloping signature")
		})
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

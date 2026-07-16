package xmldsig1_test

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// Regression for #1220: SignEnveloping must resolve a same-document reference
// (URI="#id") that points INTO the Signature's own <Object> content — here a
// <Manifest Id="manifest"> supplied via the content nodes. Before the fix the
// Signature was never attached to the document while its references were
// processed, so the "#manifest" lookup failed with ErrReferenceNotFound.
func TestSignEnveloping_ReferenceIntoOwnObject(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<Foo><Bar Id="data"><Baz Value="v"/></Bar></Foo>`)

	// Build a <ds:Manifest Id="manifest"> to place under <ds:Object>.
	manifest := doc.CreateElement("Manifest")
	require.NoError(t, manifest.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	require.NoError(t, manifest.SetLiteralAttribute("Id", "manifest"))

	ref := doc.CreateElement("Reference")
	require.NoError(t, manifest.AddChild(ref))
	require.NoError(t, ref.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	require.NoError(t, ref.SetLiteralAttribute("URI", "test.txt"))
	dm := doc.CreateElement("DigestMethod")
	require.NoError(t, ref.AddChild(dm))
	require.NoError(t, dm.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	require.NoError(t, dm.SetLiteralAttribute("Algorithm", "http://www.w3.org/2001/04/xmlenc#sha256"))
	dv := doc.CreateElement("DigestValue")
	require.NoError(t, ref.AddChild(dv))
	require.NoError(t, dv.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	h := sha256.Sum256([]byte("hello world"))
	require.NoError(t, dv.AddChild(doc.CreateText([]byte(base64.StdEncoding.EncodeToString(h[:])))))

	signer := xmldsig1.NewSigner().
		CanonicalizationMethod(xmldsig1.ExcC14N10).
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#data",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#manifest",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

	sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{manifest}, key)
	require.NoError(t, err, "signing a reference into the Signature's own Object must succeed")

	// The returned Signature is detached; the manifest lives under its Object.
	require.Nil(t, sigElem.Parent())

	// A SignedInfo reference to "#manifest" must be present with a digest.
	var haveManifestRef bool
	var walk func(n helium.Node)
	walk = func(n helium.Node) {
		if e, ok := n.(*helium.Element); ok {
			if e.LocalName() == "Reference" {
				if uri, ok := e.GetAttribute("URI"); ok && uri == "#manifest" {
					haveManifestRef = true
				}
			}
			for c := e.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
	}
	walk(sigElem)
	require.True(t, haveManifestRef, "SignedInfo must carry a Reference to #manifest")

	// The whole signature must verify once placed into the document.
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err, "signature covering a reference into its own Object must verify")
}

// Parity guard for #1220: an enveloping reference to the document element
// (URI="#root", where root IS the document element) must be digested over
// root's own subtree, unchanged — the Signature is never inserted into the
// document. The digest must be byte-identical to exclusive-C14N(root) hashed
// directly (the pre-fix value), and the signature must verify when placed as a
// sibling of root. The rejected earlier fix inserted the Signature under the
// document element, folding it into root's subtree and changing this digest.
func TestSignEnveloping_ReferenceToDocumentElement(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<root Id="root"><child>hello</child></root>`)

	payload := doc.CreateElement("Payload")
	require.NoError(t, payload.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	require.NoError(t, payload.AddChild(doc.CreateText([]byte("data"))))

	signer := xmldsig1.NewSigner().
		CanonicalizationMethod(xmldsig1.ExcC14N10).
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#root",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

	sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
	require.NoError(t, err)
	require.Nil(t, sigElem.Parent())

	// Independent expected digest: exclusive C14N of the original document
	// (whose only element is root), sha256, base64. This is the pre-fix value —
	// root with no Signature inside it.
	fresh := mustParseXML(t, `<root Id="root"><child>hello</child></root>`)
	canonical, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).CanonicalizeTo(fresh)
	require.NoError(t, err)
	sum := sha256.Sum256(canonical)
	wantDigest := base64.StdEncoding.EncodeToString(sum[:])
	require.Equal(t, wantDigest, digestValueForURI(t, sigElem, "#root"),
		"digest for #root must be byte-identical to exclusive-C14N(root) with no Signature inside")

	// Verify in a valid placement: root and Signature as siblings under a
	// wrapper, so root's subtree matches sign time.
	vdoc := mustParseXML(t, `<Wrapper><root Id="root"><child>hello</child></root></Wrapper>`)
	require.NoError(t, vdoc.DocumentElement().AddChild(sigElem))
	sigElem.SetTreeDoc(vdoc)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), vdoc)
	require.NoError(t, err, "enveloping ref to the document element must verify in a sibling placement")
}

// An id that exists BOTH in the caller's document and in the Signature's own
// Object content is an ambiguous cross-tree collision and must be rejected
// rather than silently resolving to one of them.
func TestSignEnveloping_AmbiguousIDAcrossTrees(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<Foo><Bar Id="dup"><Baz Value="v"/></Bar></Foo>`)

	manifest := doc.CreateElement("Manifest")
	require.NoError(t, manifest.SetActiveNamespace("ds", xmldsig1.NamespaceDSig))
	require.NoError(t, manifest.SetLiteralAttribute("Id", "dup"))

	signer := xmldsig1.NewSigner().
		CanonicalizationMethod(xmldsig1.ExcC14N10).
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#dup",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}).
		KeyInfo(xmldsig1.RSAKeyValueKeyInfo())

	_, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{manifest}, key)
	require.ErrorIs(t, err, xmldsig1.ErrAmbiguousReference,
		"an id present in both the document and the Signature's Object must be rejected")
}

// digestValueForURI returns the base64 DigestValue text of the Reference whose
// URI attribute equals uri, searched within sigElem's subtree.
func digestValueForURI(t *testing.T, sigElem *helium.Element, uri string) string {
	t.Helper()
	var found string
	var walk func(n helium.Node)
	walk = func(n helium.Node) {
		e, ok := n.(*helium.Element)
		if !ok {
			return
		}
		if e.LocalName() == "Reference" {
			if u, ok := e.GetAttribute("URI"); ok && u == uri {
				for c := e.FirstChild(); c != nil; c = c.NextSibling() {
					if ce, ok := c.(*helium.Element); ok && ce.LocalName() == "DigestValue" {
						found = string(ce.FirstChild().Content())
					}
				}
			}
		}
		for c := e.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(sigElem)
	return found
}

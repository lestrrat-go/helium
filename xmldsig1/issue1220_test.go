package xmldsig1_test

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	helium "github.com/lestrrat-go/helium"
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

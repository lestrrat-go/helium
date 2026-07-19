package xmldsig1_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// findByLocalNameAndID walks the doc element subtree, finds an element
// whose name local part matches localName carrying an Id/ID attribute
// equal to id, and returns the *first* such element. Test helper only.
func findByLocalNameAndID(doc *helium.Document, localName, id string) *helium.Element {
	var found *helium.Element
	var walk func(n helium.Node)
	walk = func(n helium.Node) {
		if found != nil {
			return
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return
		}
		name := elem.Name()
		l := name
		if _, after, ok := strings.Cut(name, ":"); ok {
			l = after
		}
		if l == localName {
			for _, a := range elem.Attributes() {
				if (a.Name() == "Id" || a.Name() == "ID") && a.Value() == id {
					found = elem
					return
				}
			}
		}
		for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(doc.DocumentElement())
	return found
}

// TestVerifyResult covers the VerifyResult accessors and element exposure.
func TestVerifyResult(t *testing.T) {
	// accessors covers VerifyResult.SignedElement and Covers, including their
	// nil-receiver and miss branches.
	t.Run("accessors", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		res, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.Equal(t, root, res.SignedElement(""))
		require.Nil(t, res.SignedElement("#nope"))
		require.True(t, res.Covers(root))

		other, err := doc.CreateElement("other")
		require.NoError(t, err)
		require.False(t, res.Covers(other))
		require.False(t, res.Covers(nil))

		var nilRes *xmldsig1.VerifyResult
		require.Nil(t, nilRes.SignedElement(""))
		require.False(t, nilRes.Covers(root))
	})

	// exposes signed element asserts that Verify returns the resolved element
	// pointer for each Reference, so the caller can compare pointer equality
	// against the element they are about to consume.
	t.Run("exposes signed element", func(t *testing.T) {
		xml := `<root><data Id="payload">secret</data></root>`
		key := generateRSAKey(t)
		doc := mustParseXML(t, xml)

		target := findByLocalNameAndID(doc, "data", "payload")
		require.NotNil(t, target)

		ref := xmldsig1.ReferenceConfig{
			URI:             "#payload",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		}

		sigElem, err := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(ref).
			SignDetached(t.Context(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		res, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.References, 1)
		require.Equal(t, "#payload", res.References[0].URI)
		require.Same(t, target, res.References[0].Element)
		require.Same(t, target, res.SignedElement("#payload"))
		require.True(t, res.Covers(target))
	})
}

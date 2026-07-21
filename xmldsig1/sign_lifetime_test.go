package xmldsig1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// A successful detached/enveloping sign canonicalizes the live Signature through
// a throwaway document (canonicalizeDetachedSubtree). That cross-document graft
// runs noteCrossDocumentEscape, which sets doc.slabEscaped, so the later
// doc.Free() is a no-op and never recycles the chunks backing the returned
// Signature. These tests exercise the observable consequence from outside the
// package (slabEscaped is unexported): the returned Signature stays intact and
// serializes byte-identically after doc.Free(), even with slab-pool churn in
// between that would overwrite the chunks had Free actually recycled them.
//
// churnSlabPool parses and frees several fresh documents. If Free had returned
// the signed document's chunks to the pool, these parses would reuse and
// overwrite them, corrupting a still-held returned node.
func churnSlabPool(t *testing.T) {
	t.Helper()
	for range 8 {
		d, err := helium.NewParser().Parse(t.Context(), []byte(`<a><b id="x">filler-content-to-fill-slab-chunks</b><c>more-filler</c></a>`))
		require.NoError(t, err)
		d.Free()
	}
}

func requireSignatureIntact(t *testing.T, sigElem *helium.Element) {
	t.Helper()
	require.Equal(t, "Signature", sigElem.LocalName())
	var order []string
	for c := sigElem.FirstChild(); c != nil; c = c.NextSibling() {
		if e, ok := c.(*helium.Element); ok {
			order = append(order, e.LocalName())
		}
	}
	require.Contains(t, order, "SignedInfo")
	require.Contains(t, order, "SignatureValue")
}

func TestSignDetachedSurvivesDocFree(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<root><data Id="mydata">Hello</data></root>`)

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             refURIMyData,
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})

	sigElem, err := signer.SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NotNil(t, sigElem)

	before, err := helium.WriteString(sigElem)
	require.NoError(t, err)
	require.NotEmpty(t, before)

	// A successful detached sign has already escaped doc's slab, so this Free is a
	// no-op and must NOT recycle the chunks backing sigElem.
	doc.Free()
	churnSlabPool(t)

	after, err := helium.WriteString(sigElem)
	require.NoError(t, err)
	require.Equal(t, before, after, "returned Signature must survive doc.Free() unchanged")
	requireSignatureIntact(t, sigElem)
}

func TestSignEnvelopingSurvivesDocFree(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, `<root><data Id="d1">covered</data></root>`)

	payload, err := doc.CreateElement("Payload")
	require.NoError(t, err)
	require.NoError(t, payload.AddChild(doc.CreateText([]byte("hello"))))

	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#d1",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})

	sigElem, err := signer.SignEnveloping(t.Context(), doc, []helium.Node{payload}, key)
	require.NoError(t, err)
	require.NotNil(t, sigElem)

	before, err := helium.WriteString(sigElem)
	require.NoError(t, err)
	require.NotEmpty(t, before)

	// A successful enveloping sign has already escaped doc's slab, so this Free is
	// a no-op and must NOT recycle the chunks backing sigElem.
	doc.Free()
	churnSlabPool(t)

	after, err := helium.WriteString(sigElem)
	require.NoError(t, err)
	require.Equal(t, before, after, "returned Signature must survive doc.Free() unchanged")
	requireSignatureIntact(t, sigElem)
}

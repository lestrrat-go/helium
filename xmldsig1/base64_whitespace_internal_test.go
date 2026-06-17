package xmldsig1

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// findFirstElement returns the first descendant Element matching localName.
func findFirstElement(t *testing.T, n helium.Node, name string) *helium.Element {
	t.Helper()
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if e, ok := helium.AsNode[*helium.Element](child); ok {
			if localName(e) == name {
				return e
			}
			if found := findFirstElement(t, e, name); found != nil {
				return found
			}
		}
	}
	return nil
}

// wrapText interleaves XML whitespace (CR, LF, tab, space) into s. Go's base64
// decoder skips CR/LF but rejects space and tab, so the space/tab variants are
// what exercise the strip-before-decode fix.
func wrapText(s string, col int) string {
	seps := []string{"\n", "  \t", "\r\n\t", " "}
	var out []byte
	sepIdx := 0
	for i := range len(s) {
		if i > 0 && i%col == 0 {
			out = append(out, seps[sepIdx%len(seps)]...)
			sepIdx++
		}
		out = append(out, s[i])
	}
	return string(out)
}

// setText removes existing text children and sets a single text node.
func setText(t *testing.T, e *helium.Element, text string) {
	t.Helper()
	for child := e.FirstChild(); child != nil; {
		next := child.NextSibling()
		if mn, ok := child.(helium.MutableNode); ok {
			helium.UnlinkNode(mn)
		}
		child = next
	}
	doc := e.OwnerDocument()
	require.NoError(t, e.AddChild(doc.CreateText([]byte(text))))
}

// TestVerifyLineWrappedDigestValue ensures a DigestValue carrying embedded
// whitespace (line-wrapped base64) still verifies. Because DigestValue lives
// inside SignedInfo, the wrapping must be present when SignedInfo is signed —
// c14n preserves the whitespace in the signed bytes, and the verifier must
// strip it before base64-decoding to recompute the reference digest.
func TestVerifyLineWrappedDigestValue(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const samlAssertion = `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc123" IssueInstant="2024-01-01T00:00:00Z" Version="2.0"><saml:Issuer>https://idp.example.com</saml:Issuer></saml:Assertion>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(samlAssertion))
	require.NoError(t, err)

	signer := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	sigElem := findFirstElement(t, doc, "Signature")
	require.NotNil(t, sigElem)
	signedInfo := findFirstElement(t, sigElem, "SignedInfo")
	require.NotNil(t, signedInfo)
	digestValue := findFirstElement(t, signedInfo, "DigestValue")
	require.NotNil(t, digestValue)
	sigValue := findFirstElement(t, sigElem, "SignatureValue")
	require.NotNil(t, sigValue)

	// Line-wrap the DigestValue text in place, then re-sign SignedInfo so the
	// SignatureValue covers the wrapped DigestValue.
	wrapped := wrapText(textContent(digestValue), 16)
	require.Contains(t, wrapped, " ")
	setText(t, digestValue, wrapped)

	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical)
	require.NoError(t, err)
	setText(t, sigValue, base64.StdEncoding.EncodeToString(sigBytes))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

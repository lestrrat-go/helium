package xmldsig1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// directChildLocalNames returns the local names of an element's direct element
// children, so a test can assert the caller document element gained no stray
// Signature child.
func directChildLocalNames(t *testing.T, elem *helium.Element) []string {
	t.Helper()
	var names []string
	for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
		if e, ok := helium.AsNode[*helium.Element](c); ok {
			names = append(names, e.LocalName())
		}
	}
	return names
}

// TestComputeSignedInfoDetachedProxyMatchesAttach is the byte-parity proof for
// fu10. computeAndSetSignatureValue canonicalizes SignedInfo through the
// throwaway-document proxy (canonicalizeDetachedSubtree) instead of temporarily
// attaching the detached Signature under doc.DocumentElement(). The digest, and
// therefore the SignatureValue, must be byte-identical to the attach path. This
// test canonicalizes the SAME live SignedInfo BOTH ways and asserts equal bytes
// across every C14N method and across document elements that carry namespaces
// and inherited xml:* attributes (where Canonical XML 1.0 inherits every xml:*
// including xml:id, and 1.1 inherits only xml:lang/xml:space and joins xml:base).
func TestComputeSignedInfoDetachedProxyMatchesAttach(t *testing.T) {
	docs := []struct {
		name string
		xml  string
	}{
		{
			name: "namespaces-only",
			xml:  `<a:Root xmlns:a="urn:a" xmlns:b="urn:b"><a:Child>x</a:Child></a:Root>`,
		},
		{
			name: "inherited-xml-attrs",
			xml:  `<a:Root xmlns:a="urn:a" xmlns:b="urn:b" xml:lang="en" xml:space="preserve" xml:base="http://example.com/base/" xml:id="rootid"><a:Child>x</a:Child></a:Root>`,
		},
	}
	methods := []string{ExcC14N10, C14N10, C14N11URI}

	for _, d := range docs {
		for _, method := range methods {
			t.Run(d.name+"/"+method, func(t *testing.T) {
				doc, err := helium.NewParser().Parse(t.Context(), []byte(d.xml))
				require.NoError(t, err)

				cfg := &signerConfig{c14nMethod: method, signatureAlgorithm: AlgRSASHA256}
				sig, si, _, err := buildSignatureSkeleton(doc, cfg)
				require.NoError(t, err)

				before, err := helium.WriteString(doc)
				require.NoError(t, err)

				// NEW path: canonicalize via the throwaway-document proxy. The
				// caller's document must be untouched afterward.
				gotProxy, err := canonicalizeDetachedSubtree(method, sig, si, nil)
				require.NoError(t, err)
				require.Nil(t, sig.Parent(), "Signature must remain detached after proxy canonicalization")
				afterProxy, err := helium.WriteString(doc)
				require.NoError(t, err)
				require.Equal(t, before, afterProxy, "proxy path must not mutate the caller document")

				// OLD path: temporarily attach the Signature under the document
				// element, canonicalize SignedInfo in place, then unlink.
				require.NoError(t, doc.DocumentElement().AddChild(sig))
				gotAttach, err := canonicalizeSubtree(method, si, nil)
				require.NoError(t, err)
				helium.UnlinkNode(sig)

				require.Equal(t, string(gotAttach), string(gotProxy),
					"detached-proxy SignedInfo canonicalization must be byte-identical to the attach-under-document-element path")
			})
		}
	}
}

// TestComputeAndSetSignatureValueNeverMutatesCallerDoc guards the fu10 guarantee
// that computeAndSetSignatureValue never inserts a detached Signature into the
// caller's document — on the success, error, AND panic paths.
func TestComputeAndSetSignatureValueNeverMutatesCallerDoc(t *testing.T) {
	const srcXML = `<a:Root xmlns:a="urn:a" xml:lang="en"><a:Child>x</a:Child></a:Root>`

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(srcXML))
		require.NoError(t, err)
		docElem := doc.DocumentElement()
		wantChildren := directChildLocalNames(t, docElem)

		cfg := &signerConfig{c14nMethod: ExcC14N10, signatureAlgorithm: AlgRSASHA256}
		sig, si, sv, err := buildSignatureSkeleton(doc, cfg)
		require.NoError(t, err)

		require.NoError(t, computeAndSetSignatureValue(cfg, sig, si, sv, doc, rsaKey))
		require.Nil(t, sig.Parent(), "Signature must stay detached after signing")
		require.Equal(t, wantChildren, directChildLocalNames(t, docElem),
			"caller document element must gain no stray child on the success path")
	})

	t.Run("error path leaves caller doc unchanged", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(srcXML))
		require.NoError(t, err)
		docElem := doc.DocumentElement()
		wantChildren := directChildLocalNames(t, docElem)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		// RSA signature algorithm with an ECDSA key forces signBytes to fail with
		// ErrKeyMismatch AFTER SignedInfo canonicalization has moved the detached
		// Signature into the throwaway document and moved it back.
		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		cfg := &signerConfig{c14nMethod: ExcC14N10, signatureAlgorithm: AlgRSASHA256}
		sig, si, sv, err := buildSignatureSkeleton(doc, cfg)
		require.NoError(t, err)

		err = computeAndSetSignatureValue(cfg, sig, si, sv, doc, ecKey)
		require.ErrorIs(t, err, ErrKeyMismatch)
		require.Nil(t, sig.Parent(), "Signature must stay detached after a signing error")
		require.Equal(t, wantChildren, directChildLocalNames(t, docElem),
			"caller document element must gain no stray child on the error path")
		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after, "caller document must be byte-unchanged on the error path")
	})

	t.Run("panic path leaves caller doc unchanged", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(srcXML))
		require.NoError(t, err)
		docElem := doc.DocumentElement()
		wantChildren := directChildLocalNames(t, docElem)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		cfg := &signerConfig{c14nMethod: ExcC14N10, signatureAlgorithm: AlgRSASHA256}
		sig, si, sv, err := buildSignatureSkeleton(doc, cfg)
		require.NoError(t, err)

		// Corrupt the SignedInfo subtree with a typed-nil sibling so the owner-doc
		// walk (root.SetTreeDoc) inside canonicalizeDetachedSubtree panics while the
		// live Signature is grafted into the throwaway document. Even as the panic
		// unwinds out of computeAndSetSignatureValue, the caller document must never
		// have gained the Signature.
		firstChild := si.FirstChild()
		require.NotNil(t, firstChild)
		var nilElem *helium.Element
		helium.UnsafeSetNextSibling(firstChild, nilElem)

		require.Panics(t, func() {
			_ = computeAndSetSignatureValue(cfg, sig, si, sv, doc, rsaKey)
		})
		require.Equal(t, wantChildren, directChildLocalNames(t, docElem),
			"caller document element must gain no stray child even when a panic unwinds through signing")
		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after, "caller document must be byte-unchanged even on the panic path")
	})
}

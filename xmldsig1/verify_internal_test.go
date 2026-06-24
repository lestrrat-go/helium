package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/stretchr/testify/require"
)

const dsigNS = NamespaceDSig

// findChild returns the first child element of parent with the given local
// name, failing the test if none is found.
func findChild(t *testing.T, parent *helium.Element, name string) *helium.Element {
	t.Helper()
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if domutil.LocalName(e) == name {
			return e
		}
	}
	t.Fatalf("child %q not found", name)
	return nil
}

func countChildren(parent *helium.Element, name string) int {
	n := 0
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if domutil.LocalName(e) == name {
			n++
		}
	}
	return n
}

// findFirstElement returns the first descendant Element matching localName.
func findFirstElement(t *testing.T, n helium.Node, name string) *helium.Element {
	t.Helper()
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if e, ok := helium.AsNode[*helium.Element](child); ok {
			if domutil.LocalName(e) == name {
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

func TestDigestEqual(t *testing.T) {
	require.True(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2, 3}))
	require.False(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2}))    // length mismatch
	require.False(t, digestEqual([]byte{1, 2, 3}, []byte{1, 2, 4})) // content mismatch
}

func TestParseSignatureElement(t *testing.T) {
	t.Run("missing SignedInfo", func(t *testing.T) {
		doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`"><ds:SignatureValue xmlns:ds="`+dsigNS+`">AA==</ds:SignatureValue></ds:Signature>`)
		_, err := parseSignatureElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing SignedInfo")
	})

	t.Run("missing SignatureValue", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`">`+si+`</ds:Signature>`)
		_, err := parseSignatureElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "missing SignatureValue")
	})

	t.Run("bad SignatureValue base64", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference></ds:SignedInfo>`
		doc := mustParse(t, `<ds:Signature xmlns:ds="`+dsigNS+`">`+si+
			`<ds:SignatureValue xmlns:ds="`+dsigNS+`">!!!bad</ds:SignatureValue></ds:Signature>`)
		_, err := parseSignatureElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
	})
}

func TestParseSignedInfo(t *testing.T) {
	t.Run("missing C14N algorithm", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `"/>` +
			`</ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "CanonicalizationMethod missing Algorithm")
	})

	t.Run("missing SignatureMethod algorithm", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `"/>` +
			`</ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "SignatureMethod missing Algorithm")
	})

	t.Run("no Reference", func(t *testing.T) {
		si := `<ds:SignedInfo xmlns:ds="` + dsigNS + `">` +
			`<ds:CanonicalizationMethod xmlns:ds="` + dsigNS + `" Algorithm="` + ExcC14N10 + `"/>` +
			`<ds:SignatureMethod xmlns:ds="` + dsigNS + `" Algorithm="` + AlgRSASHA256 + `"/>` +
			`</ds:SignedInfo>`
		doc := mustParse(t, si)
		var parsed parsedSignature
		err := parseSignedInfo(doc.DocumentElement(), &parsed)
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "no Reference")
	})
}

func TestParseReferenceElement(t *testing.T) {
	t.Run("missing DigestMethod algorithm", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `"/>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Contains(t, err.Error(), "DigestMethod missing Algorithm")
	})

	t.Run("bad DigestValue base64", func(t *testing.T) {
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">!!!bad</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		_, err := parseReferenceElement(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidSignature)
	})

	t.Run("with InclusiveNamespaces", func(t *testing.T) {
		const exc = ExcC14N10
		r := `<ds:Reference xmlns:ds="` + dsigNS + `" URI="">` +
			`<ds:Transforms xmlns:ds="` + dsigNS + `">` +
			`<ds:Transform xmlns:ds="` + dsigNS + `" Algorithm="` + exc + `">` +
			`<ec:InclusiveNamespaces xmlns:ec="` + exc + `" PrefixList="a b"/>` +
			`</ds:Transform></ds:Transforms>` +
			`<ds:DigestMethod xmlns:ds="` + dsigNS + `" Algorithm="` + DigestSHA256 + `"/>` +
			`<ds:DigestValue xmlns:ds="` + dsigNS + `">AA==</ds:DigestValue>` +
			`</ds:Reference>`
		doc := mustParse(t, r)
		ref, err := parseReferenceElement(doc.DocumentElement())
		require.NoError(t, err)
		require.Len(t, ref.transforms, 1)
		require.Equal(t, []string{"a", "b"}, ref.transforms[0].prefixes)
	})
}

// TestEmptyReferencesRejected guards against a SignedInfo that carries zero
// Reference children. XML-Signature requires at least one Reference; a
// SignatureValue computed over a reference-free SignedInfo cryptographically
// verifies yet covers no document content, so the signature attests to
// nothing. Verify must reject such a structure rather than report success.
//
// The attack is constructed with full key control (an attacker controlling the
// signing key is the worst case): produce a genuine signature, strip the
// Reference element, then recompute a valid SignatureValue over the now-empty
// SignedInfo. The resulting document is a perfectly valid empty-reference
// signature.
func TestEmptyReferencesRejected(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
	require.NoError(t, err)

	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		Reference(ReferenceConfig{
			URI:             "#payload",
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{ExcC14NTransform()},
		}).
		SignDetached(context.Background(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))

	// Locate SignedInfo and remove its single Reference child so the
	// SignedInfo now has zero references.
	signedInfo := findChild(t, sigElem, "SignedInfo")
	ref := findChild(t, signedInfo, "Reference")
	helium.UnlinkNode(ref)

	// Recompute a valid SignatureValue over the reference-free SignedInfo.
	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
	require.NoError(t, err)

	sigValueElem := findChild(t, sigElem, "SignatureValue")
	for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
		mc, ok := c.(helium.MutableNode)
		require.True(t, ok)
		helium.UnlinkNode(mc)
	}
	require.NoError(t, sigValueElem.AddChild(
		doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

	// Sanity: the signature value itself is cryptographically valid over the
	// empty SignedInfo, so the only thing standing between this document and a
	// false "verified" result is the empty-reference check.
	require.Equal(t, 0, countChildren(signedInfo, "Reference"))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(context.Background(), doc)
	require.Error(t, err, "signature with zero references must be rejected")
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.True(t, strings.Contains(err.Error(), "Reference"),
		"error should mention the missing Reference: %v", err)
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
	wrapped := wrapText(domutil.TextContent(digestValue), 16)
	require.Contains(t, wrapped, " ")
	setText(t, digestValue, wrapped)

	canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
	require.NoError(t, err)
	sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
	require.NoError(t, err)
	setText(t, sigValue, base64.StdEncoding.EncodeToString(sigBytes))

	verifier := NewVerifier(StaticKey(&key.PublicKey))
	_, err = verifier.Verify(t.Context(), doc)
	require.NoError(t, err)
}

// TestReferenceNamespace guards the namespace-confusion bypasses of the
// at-least-one-Reference rule: only core ds:Reference elements count.
func TestReferenceNamespace(t *testing.T) {
	// foreign namespace rejected guards against a namespace-confusion bypass of
	// the empty-reference check. The "at least one Reference" rule must count only
	// ds:Reference elements in the XML-Signature namespace. If parseSignedInfo
	// matched on local name alone, a SignedInfo carrying zero genuine ds:Reference
	// children plus a single foreign-namespace <evil:Reference> would satisfy
	// len(references) > 0 and verify against a recomputed SignatureValue while
	// covering no document content — re-opening the no-content-signature bypass.
	t.Run("foreign namespace rejected", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
		require.NoError(t, err)

		sigElem, err := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             "#payload",
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{ExcC14NTransform()},
			}).
			SignDetached(context.Background(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		signedInfo := findChild(t, sigElem, "SignedInfo")
		ref := findChild(t, signedInfo, "Reference")

		// Move the genuine ds:Reference into a foreign namespace so that, by local
		// name alone, the SignedInfo still appears to contain a Reference while it
		// no longer carries any ds:Reference.
		const evilNS = "urn:example:evil"
		require.NoError(t, ref.DeclareNamespace("evil", evilNS))
		require.NoError(t, ref.SetActiveNamespace("evil", evilNS))
		require.Equal(t, evilNS, elementNamespaceURI(ref))

		// Recompute a valid SignatureValue over the mutated SignedInfo so the only
		// thing standing between this document and a false "verified" result is the
		// namespace check on the Reference count.
		canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
		require.NoError(t, err)
		sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
		require.NoError(t, err)

		sigValueElem := findChild(t, sigElem, "SignatureValue")
		for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
			mc, ok := c.(helium.MutableNode)
			require.True(t, ok)
			helium.UnlinkNode(mc)
		}
		require.NoError(t, sigValueElem.AddChild(
			doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.Error(t, err, "signature whose only Reference is in a foreign namespace must be rejected")
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.True(t, strings.Contains(err.Error(), "Reference"),
			"error should mention the missing Reference: %v", err)
	})

	// dsig11 namespace rejected guards the core/1.1 namespace split. The
	// XML-Signature 1.1 namespace (http://www.w3.org/2009/xmldsig11#) is only for
	// new 1.1-specific elements (e.g. ECKeyValue), NOT an alternate spelling of the
	// core Reference. A dsig11:Reference therefore must not count toward the
	// at-least-one-Reference rule, so mutating the only ds:Reference into the 1.1
	// namespace must make verification fail just like a foreign-namespace Reference.
	t.Run("dsig11 namespace rejected", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
		require.NoError(t, err)

		sigElem, err := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             "#payload",
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{ExcC14NTransform()},
			}).
			SignDetached(context.Background(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		signedInfo := findChild(t, sigElem, "SignedInfo")
		ref := findChild(t, signedInfo, "Reference")

		// Move the genuine ds:Reference into the XML-Signature 1.1 namespace. By
		// local name alone the SignedInfo still appears to contain a Reference, but
		// xmldsig11# is not the core namespace, so it must not be accepted as one.
		require.NoError(t, ref.DeclareNamespace("dsig11", NamespaceDSig11))
		require.NoError(t, ref.SetActiveNamespace("dsig11", NamespaceDSig11))
		require.Equal(t, NamespaceDSig11, elementNamespaceURI(ref))

		// Recompute a valid SignatureValue over the mutated SignedInfo so the only
		// thing standing between this document and a false "verified" result is the
		// core-namespace check on the Reference count.
		canonical, err := canonicalizeSubtree(ExcC14N10, signedInfo, nil)
		require.NoError(t, err)
		sigBytes, err := signBytes(AlgRSASHA256, key, canonical, false)
		require.NoError(t, err)

		sigValueElem := findChild(t, sigElem, "SignatureValue")
		for c := sigValueElem.FirstChild(); c != nil; c = sigValueElem.FirstChild() {
			mc, ok := c.(helium.MutableNode)
			require.True(t, ok)
			helium.UnlinkNode(mc)
		}
		require.NoError(t, sigValueElem.AddChild(
			doc.CreateText([]byte(base64.StdEncoding.EncodeToString(sigBytes)))))

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.Error(t, err, "signature whose only Reference is in the xmldsig11# namespace must be rejected")
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.True(t, strings.Contains(err.Error(), "Reference"),
			"error should mention the missing Reference: %v", err)
	})

	// genuine reference still verifies is the positive control: a normal
	// ds:Reference signature must continue to verify after the namespace guard is
	// added, so the fix rejects only foreign-namespace References.
	t.Run("genuine reference still verifies", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><data Id="payload">secret</data></root>`))
		require.NoError(t, err)

		sigElem, err := NewSigner().
			SignatureAlgorithm(AlgRSASHA256).
			Reference(ReferenceConfig{
				URI:             "#payload",
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{ExcC14NTransform()},
			}).
			SignDetached(context.Background(), doc, key)
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(sigElem))

		verifier := NewVerifier(StaticKey(&key.PublicKey))
		_, err = verifier.Verify(context.Background(), doc)
		require.NoError(t, err, "a genuine ds:Reference signature must still verify")
	})
}

// TestCloneNilConfig covers the cfg == nil branches of the builder clone paths.
func TestCloneNilConfig(t *testing.T) {
	// signer covers the s.cfg == nil branch of Signer.clone by invoking a builder
	// method on a zero-value Signer.
	t.Run("signer", func(t *testing.T) {
		var s Signer // zero value: cfg is nil
		s2 := s.SignatureAlgorithm(AlgRSASHA256)
		require.NotNil(t, s2.cfg)
		require.Equal(t, AlgRSASHA256, s2.cfg.signatureAlgorithm)
		require.Equal(t, ExcC14N10, s2.cfg.c14nMethod)
	})

	// verifier covers the v.cfg == nil branch of Verifier.clone.
	t.Run("verifier", func(t *testing.T) {
		var v Verifier // zero value: cfg is nil
		v2 := v.AllowSHA1(true)
		require.NotNil(t, v2.cfg)
		require.True(t, v2.cfg.allowSHA1)
	})
}

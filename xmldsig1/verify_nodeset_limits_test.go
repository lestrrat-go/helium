package xmldsig1_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// namespaceHeavyDoc builds an unsigned document whose root carries decls
// namespace declarations and whose canonicalized subtree holds pad elements. The
// two dimensions are independent, so a node set that gave every element the
// whole in-scope namespace axis would allocate decls*pad namespace nodes for a
// document whose own size is only decls+pad.
//
// signedInfoPad places the padding inside ds:SignedInfo, the subtree
// Verifier.Verify canonicalizes before it checks the SignatureValue. Otherwise
// the padding sits in a <bulk> element OUTSIDE the ds:Signature, reached by a
// ds:KeyInfo/ds:RetrievalMethod that names it by id and carries a
// canonicalization transform — also before the SignatureValue is checked, and
// out of reach of any size check scoped to the Signature.
func namespaceHeavyDoc(decls, pad int, signedInfoPad bool) string {
	var b strings.Builder
	b.WriteString(`<root`)
	for i := range decls {
		fmt.Fprintf(&b, ` xmlns:p%d="urn:example:ns:%d"`, i, i)
	}
	b.WriteString(`>`)
	b.WriteString(`<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo>`)
	b.WriteString(`<ds:CanonicalizationMethod Algorithm="` + xmldsig1.C14N10 + `"/>`)
	b.WriteString(`<ds:SignatureMethod Algorithm="` + xmldsig1.AlgRSASHA256 + `"/>`)
	b.WriteString(`<ds:Reference URI=""><ds:DigestMethod Algorithm="` + xmldsig1.DigestSHA256 + `"/>`)
	b.WriteString(`<ds:DigestValue>AAAA</ds:DigestValue></ds:Reference>`)
	if signedInfoPad {
		// The padding is wrapped, and never left as a direct SignedInfo child, so
		// this document measures the canonicalization node set alone.
		b.WriteString(`<pad>`)
		b.WriteString(strings.Repeat(`<e a="1"/>`, pad))
		b.WriteString(`</pad>`)
	}
	b.WriteString(`</ds:SignedInfo><ds:SignatureValue>AAAA</ds:SignatureValue>`)
	if !signedInfoPad {
		b.WriteString(`<ds:KeyInfo><ds:RetrievalMethod URI="#bulk" Type="` + xmldsig1.TypeRawX509Certificate + `">`)
		b.WriteString(`<ds:Transforms><ds:Transform Algorithm="` + xmldsig1.C14N10 + `"/></ds:Transforms>`)
		b.WriteString(`</ds:RetrievalMethod></ds:KeyInfo>`)
	}
	b.WriteString(`</ds:Signature>`)
	if !signedInfoPad {
		b.WriteString(`<bulk Id="bulk">` + strings.Repeat(`<e a="1"/>`, pad) + `</bulk>`)
	}
	b.WriteString(`</root>`)
	return b.String()
}

// verifyAllocatedBytes reports how many bytes one Verify allocates. Verification
// is expected to fail — the documents carry a placeholder SignatureValue — so
// what is measured is the work Verify does before it can tell.
func verifyAllocatedBytes(t *testing.T, verifier xmldsig1.Verifier, src string) uint64 {
	t.Helper()
	doc := mustParseXML(t, src)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := verifier.Verify(t.Context(), doc)
	runtime.ReadMemStats(&after)
	require.Error(t, err)
	return after.TotalAlloc - before.TotalAlloc
}

// TestVerifyNamespaceHeavyDocument bounds the memory an unsigned,
// namespace-dense document can force out of verification before the
// SignatureValue is checked. The node set built for canonicalization is linear
// in the document, so the cost is bounded against the document's own SIZE rather
// than against a second measurement of the same code.
//
// 96 bytes of allocation per input byte is far above what these documents
// actually cost and far below the hundreds of megabytes a namespace axis
// materialized per element allocates for the same input.
func TestVerifyNamespaceHeavyDocument(t *testing.T) {
	const allocPerInputByte = 96

	key := generateRSAKey(t)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))

	// SignedInfo is canonicalized before the SignatureValue is checked, so its
	// subtree is attacker-controlled work that needs no key and no valid digest.
	t.Run("SignedInfo subtree", func(t *testing.T) {
		src := namespaceHeavyDoc(1350, 3500, true)
		allocated := verifyAllocatedBytes(t, verifier, src)
		require.Less(t, allocated, uint64(allocPerInputByte*len(src)),
			"verifying a %d-byte document allocated %d bytes", len(src), allocated)
	})

	// A ds:RetrievalMethod resolves any element in the document by id, so the
	// expensive subtree can sit entirely outside the ds:Signature — a size check
	// scoped to the Signature element would not bound it. KeyInfo resolution runs
	// before the SignatureValue check, so this too needs no key.
	t.Run("RetrievalMethod subtree outside the Signature", func(t *testing.T) {
		src := namespaceHeavyDoc(1350, 3500, false)
		doc := mustParseXML(t, src)

		// The retrieved <bulk> element is not a certificate, so the failure comes
		// from KeyInfo resolution: the transform pipeline canonicalized that
		// subtree, which is the work being bounded.
		_, err := verifier.Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)

		allocated := verifyAllocatedBytes(t, verifier, src)
		require.Less(t, allocated, uint64(allocPerInputByte*len(src)),
			"verifying a %d-byte document allocated %d bytes", len(src), allocated)
	})
}

// base64RetrievalDoc points the ds:RetrievalMethod's transform at the Base64
// decode transform instead of a canonicalization one. XMLDSig core §6.6.2
// converts a node-set to octets by concatenating its text nodes — self::text()
// is empty for a namespace node, an attribute, an element, a comment and a
// processing instruction — so this transform reads no namespace at all, and the
// same document must not become two orders of magnitude more expensive for
// naming it. wholeDoc selects the URI="" form, whose selection is the whole
// document, well beyond the <bulk> subtree.
func base64RetrievalDoc(decls, pad int, wholeDoc bool) string {
	src := strings.Replace(namespaceHeavyDoc(decls, pad, false),
		`<ds:Transform Algorithm="`+xmldsig1.C14N10+`"/>`,
		`<ds:Transform Algorithm="`+xmldsig1.TransformBase64+`"/>`,
		1)
	if !wholeDoc {
		return src
	}
	return strings.Replace(src, `<ds:RetrievalMethod URI="#bulk"`, `<ds:RetrievalMethod URI=""`, 1)
}

// TestVerifyBase64NodeSetDocument bounds the memory a Base64 transform on a
// namespace-dense same-document selection can force out of verification before
// the SignatureValue is checked. The bound is the same per-input-byte figure the
// canonicalization path is held to, because Base64 reads strictly less of the
// document than canonicalization does.
func TestVerifyBase64NodeSetDocument(t *testing.T) {
	const allocPerInputByte = 96

	key := generateRSAKey(t)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))

	for _, tc := range []struct {
		name     string
		wholeDoc bool
	}{
		// The subtree form selects the <bulk> element the RetrievalMethod names by
		// id; the whole-document form selects every top-level element instead, which
		// is a separate collection path.
		{name: "RetrievalMethod subtree outside the Signature"},
		{name: "whole-document RetrievalMethod", wholeDoc: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := base64RetrievalDoc(1350, 3500, tc.wholeDoc)
			doc := mustParseXML(t, src)

			// The decoded octets are not a certificate, so the failure comes from
			// KeyInfo resolution — after the transform pipeline consumed the
			// selection, which is the work being bounded.
			_, err := verifier.Verify(t.Context(), doc)
			require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)

			allocated := verifyAllocatedBytes(t, verifier, src)
			require.Less(t, allocated, uint64(allocPerInputByte*len(src)),
				"verifying a %d-byte document allocated %d bytes", len(src), allocated)
		})
	}
}

// namespaceDensePayload is a namespace-dense document with no signature of its
// own: decls declarations on the root and pad elements below it, so a node-set
// giving every element the whole in-scope namespace axis holds decls*pad
// namespace nodes for an input of size decls+pad.
func namespaceDensePayload(decls, pad int) string {
	var b strings.Builder
	b.WriteString(`<payload`)
	for i := range decls {
		fmt.Fprintf(&b, ` xmlns:p%d="urn:example:ns:%d"`, i, i)
	}
	b.WriteString(`>`)
	b.WriteString(strings.Repeat(`<e a="1"/>`, pad))
	b.WriteString(`</payload>`)
	return b.String()
}

// base64C14NRetrievalDoc carries a namespace-dense document as base64 text and
// runs it through [Base64, C14N 1.0]: Base64 decodes the text to octets, and the
// canonicalization transform re-parses those octets into a document of its own.
// That second document is attacker-supplied too and is reached before the
// SignatureValue is checked, so the node-set built for it needs the same bound
// the original document's does.
func base64C14NRetrievalDoc(decls, pad int) string {
	payload := base64.StdEncoding.EncodeToString([]byte(namespaceDensePayload(decls, pad)))

	var b strings.Builder
	b.WriteString(`<root><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo>`)
	b.WriteString(`<ds:CanonicalizationMethod Algorithm="` + xmldsig1.C14N10 + `"/>`)
	b.WriteString(`<ds:SignatureMethod Algorithm="` + xmldsig1.AlgRSASHA256 + `"/>`)
	b.WriteString(`<ds:Reference URI="#bulk"><ds:DigestMethod Algorithm="` + xmldsig1.DigestSHA256 + `"/>`)
	b.WriteString(`<ds:DigestValue>AAAA</ds:DigestValue></ds:Reference>`)
	b.WriteString(`</ds:SignedInfo><ds:SignatureValue>AAAA</ds:SignatureValue>`)
	b.WriteString(`<ds:KeyInfo><ds:RetrievalMethod URI="#bulk" Type="` + xmldsig1.TypeRawX509Certificate + `">`)
	b.WriteString(`<ds:Transforms>`)
	b.WriteString(`<ds:Transform Algorithm="` + xmldsig1.TransformBase64 + `"/>`)
	b.WriteString(`<ds:Transform Algorithm="` + xmldsig1.C14N10 + `"/>`)
	b.WriteString(`</ds:Transforms></ds:RetrievalMethod></ds:KeyInfo></ds:Signature>`)
	b.WriteString(`<bulk Id="bulk">` + payload + `</bulk></root>`)
	return b.String()
}

// TestVerifyBase64ThenCanonicalizationDocument bounds the node-set built for a
// document parsed at an octet boundary. The consumer here takes that set whole,
// so it observes no more than the reduced, mode-aware namespace membership the
// same canonicalization reads for a document that was never re-parsed.
func TestVerifyBase64ThenCanonicalizationDocument(t *testing.T) {
	const allocPerInputByte = 96

	key := generateRSAKey(t)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))

	src := base64C14NRetrievalDoc(3000, 1000)
	doc := mustParseXML(t, src)

	// The canonical octets are not a certificate, so the failure comes from
	// KeyInfo resolution — after the re-parsed document's node set was built.
	_, err := verifier.Verify(t.Context(), doc)
	require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)

	allocated := verifyAllocatedBytes(t, verifier, src)
	require.Less(t, allocated, uint64(allocPerInputByte*len(src)),
		"verifying a %d-byte document allocated %d bytes", len(src), allocated)
}

// cancellingKeySource resolves a key and cancels the verification context as it
// does, so the work that follows key resolution — the SignedInfo
// canonicalization — runs under a cancelled context.
type cancellingKeySource struct {
	key    any
	cancel context.CancelFunc
}

func (s cancellingKeySource) ResolveKey(_ context.Context, _ *xmldsig1.KeyInfoData, _ string) (any, error) {
	s.cancel()
	return s.key, nil
}

// xpathFilterRetrievalDoc points a ds:KeyInfo/ds:RetrievalMethod at a subtree
// outside the ds:Signature through an XPath filter transform. That transform's
// input node-set carries the complete namespace axis on every element — the
// filter is evaluated once per node and may drop an interior element — so the
// set is quadratic in the document by the transform's own data model, and
// RetrievalMethod transforms run before the SignatureValue is checked. A
// deadline is what bounds it, which requires the walk to poll the context.
func xpathFilterRetrievalDoc(decls, pad int) string {
	src := namespaceHeavyDoc(decls, pad, false)
	return strings.Replace(src,
		`<ds:Transform Algorithm="`+xmldsig1.C14N10+`"/>`,
		`<ds:Transform Algorithm="`+xmldsig1.TransformXPath+`"><ds:XPath>true()</ds:XPath></ds:Transform>`,
		1)
}

// xpathFilterInheritedTargetDoc keeps the large namespace axis on the selected
// target's ancestor, outside ds:Signature. This isolates the allocation needed
// to seed the RetrievalMethod selection from unrelated signature namespace
// processing.
func xpathFilterInheritedTargetDoc(decls int) string {
	var b strings.Builder
	b.WriteString(`<root><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo>`)
	b.WriteString(`<ds:CanonicalizationMethod Algorithm="` + xmldsig1.C14N10 + `"/>`)
	b.WriteString(`<ds:SignatureMethod Algorithm="` + xmldsig1.AlgRSASHA256 + `"/>`)
	b.WriteString(`<ds:Reference URI=""><ds:DigestMethod Algorithm="` + xmldsig1.DigestSHA256 + `"/>`)
	b.WriteString(`<ds:DigestValue>AAAA</ds:DigestValue></ds:Reference></ds:SignedInfo>`)
	b.WriteString(`<ds:SignatureValue>AAAA</ds:SignatureValue><ds:KeyInfo>`)
	b.WriteString(`<ds:RetrievalMethod URI="#bulk" Type="` + xmldsig1.TypeRawX509Certificate + `">`)
	b.WriteString(`<ds:Transforms><ds:Transform Algorithm="` + xmldsig1.TransformXPath + `">`)
	b.WriteString(`<ds:XPath>true()</ds:XPath></ds:Transform></ds:Transforms>`)
	b.WriteString(`</ds:RetrievalMethod></ds:KeyInfo></ds:Signature><scope`)
	for i := range decls {
		fmt.Fprintf(&b, ` xmlns:p%d="urn:example:ns:%d"`, i, i)
	}
	b.WriteString(`><bulk Id="bulk"/></scope></root>`)
	return b.String()
}

// xpathFilterRetrievalMembers is the exact number of members in the <bulk>
// subtree's XPath input node-set: the bulk element, its Id attribute and
// inherited namespace axis, then each child element, attribute, and complete
// namespace axis.
func xpathFilterRetrievalMembers(decls, pad int) int {
	return 2 + decls + pad*(decls+2)
}

func TestVerifyXPathFilterNodeLimit(t *testing.T) {
	const (
		decls = 3
		pad   = 4
	)
	key := generateRSAKey(t)
	src := xpathFilterRetrievalDoc(decls, pad)
	limit := xpathFilterRetrievalMembers(decls, pad)

	t.Run("exact boundary", func(t *testing.T) {
		doc := mustParseXML(t, src)
		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxXPathFilterNodes(limit).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrInvalidKeyInfo)
		require.NotErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)
	})

	t.Run("one member over", func(t *testing.T) {
		doc := mustParseXML(t, src)
		ks := &recordingKeySource{key: &key.PublicKey}
		_, err := xmldsig1.NewVerifier(ks).
			MaxXPathFilterNodes(limit-1).
			Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)
		require.Contains(t, err.Error(), fmt.Sprintf("exceeds %d node-set members", limit-1))
		require.False(t, ks.called, "the XPath input limit must fire before key resolution")
	})

	t.Run("cancelled context wins", func(t *testing.T) {
		doc := mustParseXML(t, src)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			MaxXPathFilterNodes(limit-1).
			Verify(ctx, doc)
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)
	})
}

// TestVerifyXPathFilterNodeLimitAllocation keeps the default bound tied to its
// purpose: this compact namespace-heavy document would otherwise materialize
// over 1.6 million namespace-axis members before authentication. The default
// refuses it after 65,536 members without allocating the rejected wrapper.
func TestVerifyXPathFilterNodeLimitAllocation(t *testing.T) {
	const maxVerifyAllocation = 32 << 20

	key := generateRSAKey(t)
	src := xpathFilterRetrievalDoc(800, 2000)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
	for _, candidate := range []xmldsig1.Verifier{verifier, verifier.MaxXPathFilterNodes(0)} {
		doc := mustParseXML(t, src)
		_, err := candidate.Verify(t.Context(), doc)
		require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)
	}

	allocated := verifyAllocatedBytes(t, verifier, src)
	t.Logf("default XPath input limit allocated %d bytes for a %d-byte fixture", allocated, len(src))
	require.Less(t, allocated, uint64(maxVerifyAllocation),
		"verifying a %d-byte namespace-heavy document allocated %d bytes", len(src), allocated)
}

// TestVerifyXPathFilterNodeLimitBoundsInheritedScope pins that the member cap
// applies while the subtree root's inherited namespace scope is seeded. With a
// one-member limit, the root consumes the entire budget, so verification must
// reject the first namespace without building a map of every declaration.
func TestVerifyXPathFilterNodeLimitBoundsInheritedScope(t *testing.T) {
	const (
		decls               = 20_000
		maxVerifyAllocation = 2 << 20
	)

	key := generateRSAKey(t)
	src := xpathFilterInheritedTargetDoc(decls)
	verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
		MaxXPathFilterNodes(1)
	doc := mustParseXML(t, src)
	_, err := verifier.Verify(t.Context(), doc)
	require.ErrorIs(t, err, xmldsig1.ErrResourceLimitExceeded)

	allocated := verifyAllocatedBytes(t, verifier, src)
	t.Logf("limited inherited namespace scope allocated %d bytes for %d declarations", allocated, decls)
	require.Less(t, allocated, uint64(maxVerifyAllocation),
		"verifying a document with %d inherited declarations allocated %d bytes", decls, allocated)
}

// TestVerifyDeadlineDuringNodeSetConstruction pins that a deadline stops node-set
// construction while it runs, well before the whole set is built. The
// document costs seconds of work to canonicalize, so a deadline an order of
// magnitude shorter must surface as the deadline error.
func TestVerifyDeadlineDuringNodeSetConstruction(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, xpathFilterRetrievalDoc(800, 2000))

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
		MaxXPathFilterNodes(-1).
		Verify(ctx, doc)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestVerifyCancelledDuringCanonicalization pins that the SignedInfo
// canonicalization honors a cancelled context. Cancellation arrives from the
// KeySource, which Verify calls immediately before canonicalizing SignedInfo, so
// the walk starts already cancelled and must abandon the subtree instead of
// canonicalizing it and reporting an unrelated signature failure.
func TestVerifyCancelledDuringCanonicalization(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, namespaceHeavyDoc(4, 400, true))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, err := xmldsig1.NewVerifier(cancellingKeySource{key: &key.PublicKey, cancel: cancel}).
		Verify(ctx, doc)
	require.ErrorIs(t, err, context.Canceled)
}

package xmldsig1_test

import (
	"context"
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
		// The padding is wrapped rather than left as a direct SignedInfo child so
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

// TestVerifyDeadlineDuringNodeSetConstruction pins that a deadline stops node-set
// construction rather than only being noticed once the whole set is built. The
// document costs seconds of work to canonicalize, so a deadline an order of
// magnitude shorter must surface as the deadline error.
func TestVerifyDeadlineDuringNodeSetConstruction(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, xpathFilterRetrievalDoc(800, 2000))

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	_, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).Verify(ctx, doc)
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

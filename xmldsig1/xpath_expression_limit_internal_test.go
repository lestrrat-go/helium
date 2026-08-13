package xmldsig1

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"runtime"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// xpathFilterExpressionOfLength returns a syntactically valid XPath filter
// expression of exactly n bytes: a single name test, which is the cheapest shape
// to compile, so a boundary case measures the ceiling and nothing else.
func xpathFilterExpressionOfLength(n int) string {
	const axis = "self::"
	return axis + strings.Repeat("a", n-len(axis))
}

// xpathFilterSignatureDoc builds a document carrying a ds:Signature whose single
// same-document Reference declares an XPath filter transform holding expr,
// followed by a canonicalization transform. DigestValue and SignatureValue are
// placeholders: every caller here is about what verification does with the
// expression before it reaches the SignatureValue check.
func xpathFilterSignatureDoc(t *testing.T, expr string) *helium.Document {
	t.Helper()
	return mustParse(t, `<root xmlns:t="urn:t" xmlns:ds="http://www.w3.org/2000/09/xmldsig#">`+
		`<t:data>KEEPVAL</t:data>`+
		`<ds:Signature>`+
		`<ds:SignedInfo>`+
		`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>`+
		`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>`+
		`<ds:Reference URI="">`+
		`<ds:Transforms>`+
		`<ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116">`+
		`<ds:XPath>`+expr+`</ds:XPath>`+
		`</ds:Transform>`+
		`<ds:Transform Algorithm="http://www.w3.org/2006/12/xml-c14n11"/>`+
		`</ds:Transforms>`+
		`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>`+
		`<ds:DigestValue>AA==</ds:DigestValue>`+
		`</ds:Reference>`+
		`</ds:SignedInfo>`+
		`<ds:SignatureValue>AA==</ds:SignatureValue>`+
		`</ds:Signature>`+
		`</root>`)
}

// TestXPathFilterExpressionCeiling pins the fixed length ceiling on a
// ds:Transform/XPath filter expression. The expression is attacker-controlled
// and is compiled during Reference preflight, before the SignatureValue is
// checked, and compiling one costs more than linearly in its length, so the
// length is refused where the expression is read off the document rather than
// anywhere further in.
func TestXPathFilterExpressionCeiling(t *testing.T) {
	// Both cases are the same shape one byte apart, so what separates them is the
	// ceiling itself and not the expression's contents.
	t.Run("at the ceiling parses and compiles", func(t *testing.T) {
		expr := xpathFilterExpressionOfLength(maxXPathFilterExpressionBytes)
		doc := xpathFilterSignatureDoc(t, expr)
		transformElem := findLocal(doc.DocumentElement(), "Transform")
		require.NotNil(t, transformElem)

		parsedExpr, _, _, err := parseXPathTransform(transformElem)
		require.NoError(t, err)
		require.Equal(t, expr, parsedExpr)

		// An expression the ceiling accepts must still be usable: the ceiling is a
		// resource bound, not a second syntax gate.
		compiled, err := compileXPathFilterExpression(parsedExpr, newDSigXPathEvaluator(nil, nil, defaultXPathOpLimit))
		require.NoError(t, err)
		require.NotNil(t, compiled)
	})

	t.Run("one byte over the ceiling is refused before compilation", func(t *testing.T) {
		expr := xpathFilterExpressionOfLength(maxXPathFilterExpressionBytes + 1)
		doc := xpathFilterSignatureDoc(t, expr)
		transformElem := findLocal(doc.DocumentElement(), "Transform")
		require.NotNil(t, transformElem)

		_, _, _, err := parseXPathTransform(transformElem)
		require.ErrorIs(t, err, ErrResourceLimitExceeded)
		// The compiler reports a rejected expression as ErrUnsupportedTransform, so
		// its absence is what shows the expression was refused for its length and
		// never handed to the compiler.
		require.NotErrorIs(t, err, ErrUnsupportedTransform)
	})

	// The whole Signature parse carries the same verdict, so an over-length
	// expression cannot reach preflight through a Reference either.
	t.Run("over-ceiling expression fails the signature parse", func(t *testing.T) {
		doc := xpathFilterSignatureDoc(t, xpathFilterExpressionOfLength(maxXPathFilterExpressionBytes+1))
		sig := findSig(doc.DocumentElement())
		require.NotNil(t, sig)

		_, err := parseSignatureElement(t.Context(), newVerifyBudget(&verifierConfig{}), sig)
		require.ErrorIs(t, err, ErrResourceLimitExceeded)
	})
}

// TestXPathFilterExpressionErrorSize pins the size of the error a rejected
// expression produces. The expression is interpolated into the compiler's
// diagnostic, so naming the whole of one would make the error string as large as
// the expression itself for whatever a caller then logs.
func TestXPathFilterExpressionErrorSize(t *testing.T) {
	// A function call that is never closed, at the longest length the ceiling
	// admits. The XPath parser reports the unclosed call without repeating the
	// name, so what is left in the message is what this package interpolates.
	expr := strings.Repeat("a", maxXPathFilterExpressionBytes-1) + "("

	_, err := compileXPathFilterExpression(expr, newDSigXPathEvaluator(nil, nil, defaultXPathOpLimit))
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	// The bound is derived from the truncation length rather than written out, and
	// the message is asserted by size instead of by text so the wording stays free
	// to change.
	require.Less(t, len(err.Error()), maxErrorExpressionBytes+512)
	require.Contains(t, err.Error(), expr[:maxErrorExpressionBytes])
	require.NotContains(t, err.Error(), expr)
}

// TestXPathFilterShortExpressionVerifies is what proves the ceiling is not too
// tight: a realistic filter — the shape of the W3C defCan-1 interop vector,
// which is about a hundred characters — still verifies end to end through the
// public Verify.
func TestXPathFilterShortExpressionVerifies(t *testing.T) {
	// The filter keeps only the t:data subtree, so the ds:Signature is outside the
	// digested node-set and writing the DigestValue below does not change the
	// bytes that were just digested.
	doc := xpathFilterSignatureDoc(t, "ancestor-or-self::t:data")
	require.Less(t, len("ancestor-or-self::t:data"), maxXPathFilterExpressionBytes)

	sig := findSig(doc.DocumentElement())
	require.NotNil(t, sig)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	parsed, err := parseSignatureElement(t.Context(), newVerifyBudget(&verifierConfig{}), sig)
	require.NoError(t, err)
	require.Len(t, parsed.references, 1)

	cfg := &verifierConfig{keySource: StaticKey(&key.PublicKey)}
	_, canonical, _, err := canonicalizeReference(t.Context(), cfg, doc, sig, parsed.references[0])
	require.NoError(t, err)
	require.Contains(t, string(canonical), "KEEPVAL", "the filter must keep the referenced content")

	digest, err := computeDigest(parsed.references[0].digestAlgorithm, canonical, false)
	require.NoError(t, err)
	digestValueElem := findLocal(sig, "DigestValue")
	require.NotNil(t, digestValueElem)
	setElementText(t, doc, digestValueElem, base64.StdEncoding.EncodeToString(digest))

	signedInfoCanon, err := canonicalizeSubtree(t.Context(), parsed.c14nMethod, parsed.signedInfoElem, parsed.c14nPrefixes)
	require.NoError(t, err)
	sigValue, err := signBytes(parsed.signatureAlg, key, signedInfoCanon, false)
	require.NoError(t, err)
	sigValueElem := findLocal(sig, "SignatureValue")
	require.NotNil(t, sigValueElem)
	setElementText(t, doc, sigValueElem, base64.StdEncoding.EncodeToString(sigValue))

	result, err := NewVerifier(StaticKey(&key.PublicKey)).Verify(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, result.References, 1)
}

// TestXPathFilterExpressionAllocation pins what an over-ceiling expression may
// ALLOCATE, which the rejection assertions above cannot see. Compiling an XPath
// expression costs much more than the expression's own size — a chain of
// predicates allocates about a hundred times it — and the XPath parser's
// nesting-depth bound does not constrain a flat chain of any length, so a
// ceiling that were charged after the compile would be an accounting formality:
// the memory is allocated by the time the error is returned.
//
// The measurement reads the process-wide TotalAlloc delta across Verify, so
// these runs must NOT be parallel: a concurrent test's allocations would pollute
// the delta.
func TestXPathFilterExpressionAllocation(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own Verify.

	// A flat predicate chain is the amplifying shape, and it stays shallow, so the
	// nesting-depth bound never fires on it. The length is kept modest: it only
	// has to be far enough over the ceiling that a per-byte cost and a
	// compile-sized cost cannot be confused.
	const step = "[true()]"
	chain := "self::a" + strings.Repeat(step, (16<<10)/len(step))
	require.Greater(t, len(chain), maxXPathFilterExpressionBytes)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	verifier := NewVerifier(StaticKey(&key.PublicKey))

	// The baseline is the same document with a filter the ceiling accepts, so the
	// difference between the two is what the over-length expression costs beyond
	// the document that carries it.
	baseline := verifyAllocations(t, verifier, xpathFilterSignatureDoc(t, "self::a"))
	oversize := verifyAllocations(t, verifier, xpathFilterSignatureDoc(t, chain))

	// Reading the expression off the DOM costs a copy of it, and parsing the
	// document that holds it costs a few more, so the slack is a small multiple of
	// the expression rather than nothing at all. Compiling it would cost about a
	// hundred times, which no multiple this small can hide.
	slack := uint64(8 * len(chain))
	require.Less(t, oversize, baseline+slack,
		"verifying a %d-byte XPath filter expression allocated %d bytes against a %d-byte baseline", len(chain), oversize, baseline)
}

// verifyAllocations reports how many bytes one Verify of doc allocates. Every
// document these callers pass carries a placeholder SignatureValue, so Verify is
// expected to fail; what is measured is the work it does before it does.
func verifyAllocations(t *testing.T, verifier Verifier, doc *helium.Document) uint64 {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := verifier.Verify(t.Context(), doc)
	runtime.ReadMemStats(&after)
	require.Error(t, err)
	return after.TotalAlloc - before.TotalAlloc
}

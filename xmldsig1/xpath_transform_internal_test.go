package xmldsig1

import (
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// stepsFromParsed converts a parsedReference's transforms into pipeline steps,
// mirroring canonicalizeReference so a test drives the real resolution.
func stepsFromParsed(ref parsedReference) []transformStep {
	steps := make([]transformStep, len(ref.transforms))
	for i, t := range ref.transforms {
		steps[i] = transformStep(t)
	}
	return steps
}

// TestXPathTransformDefCanDigest validates the XPath filter transform against
// the W3C "defCan-1" interop vector. That vector's Reference targets an EXTERNAL
// document (URI="c14n11/xml-base-input.xml"), which resolveReference rejects
// fail-closed, so the reference cannot be dereferenced through the public API.
// The transform pipeline itself — parsing the ds:Transform/XPath expression and
// its namespace context, evaluating the filter once per node, and canonicalizing
// the surviving node-set — is exercised directly against the external document
// and must reproduce the exact DigestValue the W3C signer recorded.
func TestXPathTransformDefCanDigest(t *testing.T) {
	// The parsed XPath transform comes from the real signature file.
	_, parsed := parseVectorSignature(t, "defCan-1-signature.xml")
	require.Len(t, parsed.references, 1)
	ref := parsed.references[0]
	require.Equal(t, "c14n11/xml-base-input.xml", ref.uri,
		"defCan-1 references an external document, resolved by the harness, not the public API")

	pipe, err := resolveTransformPipeline(stepsFromParsed(ref))
	require.NoError(t, err)
	require.Len(t, pipe.xpathFilters, 1, "one XPath filter transform")
	require.Equal(t, "http://www.ietf.org", pipe.xpathFilters[0].ns["ietf"],
		"the XPath element's ietf: namespace binding must be captured")
	require.Equal(t, C14N11URI, pipe.c14nMethod)

	// The external reference document.
	extDoc := mustParseInteropFile(t, "xml-base-input.xml")
	nodes := collectDocumentNodes(extDoc)
	for _, f := range pipe.xpathFilters {
		nodes, err = applyXPathFilter(t.Context(), nodes, f)
		require.NoError(t, err)
	}
	canonical, err := canonicalizeNodeSet(pipe.c14nMethod, nodes, extDoc, pipe.prefixes)
	require.NoError(t, err)

	computed, err := computeDigest(ref.digestAlgorithm, canonical, true)
	require.NoError(t, err)
	require.True(t, digestEqual(computed, ref.digestValue),
		"defCan-1 XPath+c14n11 digest must match the signed DigestValue")
}

// TestXPathTransformThroughPipeline drives a same-document Reference carrying an
// XPath filter transform through canonicalizeReference end to end, confirming
// the filter is wired into the verify path: the kept subtree survives and the
// dropped subtree is absent from the canonical octets.
func TestXPathTransformThroughPipeline(t *testing.T) {
	const xml = `<root xmlns:t="urn:t"><t:sec t:id="x"><t:keep>KEEPVAL</t:keep><t:drop>DROPVAL</t:drop></t:sec></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	sec := findLocal(doc.DocumentElement(), "sec")
	require.NotNil(t, sec)
	// Give it an id the resolver recognizes.
	require.NoError(t, sec.SetAttribute("Id", "x"))

	// Keep every node that is not inside a t:drop element.
	ref := parsedReference{
		uri:             "#x",
		digestAlgorithm: DigestSHA256,
		transforms: []parsedTransform{
			{algorithm: TransformXPath, xpathExpr: "not(ancestor-or-self::t:drop)", xpathNS: map[string]string{"t": "urn:t"}},
			{algorithm: C14N11URI},
		},
	}

	target, canonical, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
	require.NoError(t, err)
	require.Equal(t, sec, target)
	require.Contains(t, string(canonical), "KEEPVAL", "kept subtree must survive the XPath filter")
	require.NotContains(t, string(canonical), "DROPVAL", "dropped subtree must be filtered out")
	require.NotContains(t, string(canonical), "t:drop", "dropped element must be filtered out")
}

// TestXPathTransformFailClosed locks the fail-closed edges of the XPath filter
// transform: an empty/absent expression, and the whole-document unsupported
// XSLT-transform chain from the defCan-2/3 vectors.
func TestXPathTransformFailClosed(t *testing.T) {
	t.Run("empty expression", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><d>x</d></root>`))
		require.NoError(t, err)
		ref := parsedReference{
			uri:             "",
			digestAlgorithm: DigestSHA256,
			transforms:      []parsedTransform{{algorithm: TransformXPath}},
		}
		_, canon, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Nil(t, canon)
	})

	t.Run("defCan-2 XSLT chain rejected", func(t *testing.T) {
		_, err := parseSignatureFromInterop(t, "defCan-2-signature.xml")
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("defCan-3 XSLT chain rejected", func(t *testing.T) {
		_, err := parseSignatureFromInterop(t, "defCan-3-signature.xml")
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})
}

// TestXPathTransformNamespaceContextIsolated confirms an unprefixed name test in
// an XPath transform matches only no-namespace nodes: XPath 1.0 has no default
// element namespace, so the ds:Transform/XPath default namespace is not applied.
func TestXPathTransformNamespaceContextIsolated(t *testing.T) {
	// The XPath element's default namespace is urn:t, but an unprefixed "sec"
	// name test must NOT bind to it. Keeping "not(self::sec)" therefore keeps
	// every node (nothing is in the no-namespace "sec"), so the t:sec subtree
	// survives intact.
	const xml = `<root xmlns:t="urn:t"><t:sec Id="x"><t:v>V</t:v></t:sec></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	ref := parsedReference{
		uri:             "#x",
		digestAlgorithm: DigestSHA256,
		transforms: []parsedTransform{
			{algorithm: TransformXPath, xpathExpr: "not(self::sec)", xpathNS: map[string]string{"t": "urn:t"}},
			{algorithm: C14N11URI},
		},
	}
	_, canonical, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
	require.NoError(t, err)
	require.Contains(t, string(canonical), "V")
}

func mustParseInteropFile(t *testing.T, name string) *helium.Document {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "interop", name))
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	return doc
}

func parseSignatureFromInterop(t *testing.T, name string) (*parsedSignature, error) {
	t.Helper()
	doc := mustParseInteropFile(t, name)
	sig := findSig(doc.DocumentElement())
	require.NotNil(t, sig)
	return parseSignatureElement(sig)
}

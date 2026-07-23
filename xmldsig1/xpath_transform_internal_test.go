package xmldsig1

import (
	"context"
	"os"
	"slices"
	"sync"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestXPathTransformDefCanDigest validates the ordered external-reference path
// against the W3C defCan-1 XPath + C14N 1.1 interop vector.
func TestXPathTransformDefCanDigest(t *testing.T) {
	// The parsed XPath transform comes from the real signature file.
	doc, parsed := parseVectorSignature(t, "defCan-1-signature.xml")
	require.Len(t, parsed.references, 1)
	ref := parsed.references[0]
	require.Equal(t, "c14n11/xml-base-input.xml", ref.uri,
		"defCan-1 references an external document, resolved by the harness, not the public API")

	require.Len(t, ref.transforms, 2)
	require.Equal(t, "http://www.ietf.org", ref.transforms[0].xpathNS["ietf"],
		"the XPath element's ietf: namespace binding must be captured")
	require.Equal(t, C14N11URI, ref.transforms[1].algorithm)

	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(os.DirFS("testdata/interop"))}
	canonical, err := resolveExternalReference(t.Context(), cfg, doc, ref)
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

// TestXPathTransformFailClosed locks the empty-expression validation edge.
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
}

type recordedXSLTCall struct {
	stylesheet []byte
	input      []byte
}

type recordingXSLT3Transformer struct {
	mu    sync.Mutex
	calls []recordedXSLTCall
}

func (r *recordingXSLT3Transformer) TransformXSLT(ctx context.Context, stylesheet, input []byte) ([]byte, error) {
	r.mu.Lock()
	r.calls = append(r.calls, recordedXSLTCall{stylesheet: slices.Clone(stylesheet), input: slices.Clone(input)})
	r.mu.Unlock()
	out, err := xslt3Transformer{}.TransformXSLT(ctx, stylesheet, input)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *recordingXSLT3Transformer) snapshot() []recordedXSLTCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

// TestXPathTransformDefCanMultiphase verifies the W3C defCan-2/3 vectors whose
// ordered transforms cross between node sets and octets more than once.
func TestXPathTransformDefCanMultiphase(t *testing.T) {
	cases := map[string]int{
		"defCan-2-signature.xml": 1,
		"defCan-3-signature.xml": 2,
	}
	for name, wantXSLTCalls := range cases {
		t.Run(name, func(t *testing.T) {
			doc, parsed := parseVectorSignature(t, name)
			require.Len(t, parsed.references, 1)
			ref := parsed.references[0]

			recorder := &recordingXSLT3Transformer{}
			cfg := &verifierConfig{
				allowSHA1:         true,
				referenceResolver: FSReferenceResolver(os.DirFS("testdata/interop")),
				xsltTransformer:   recorder,
			}
			_, canonical, external, err := canonicalizeReference(t.Context(), cfg, doc, findSig(doc.DocumentElement()), ref)
			require.NoError(t, err)
			require.True(t, external)
			computed, err := computeDigest(ref.digestAlgorithm, canonical, true)
			require.NoError(t, err)
			require.True(t, digestEqual(computed, ref.digestValue))
			require.Len(t, recorder.snapshot(), wantXSLTCalls)

			verifyTransformer := &recordingXSLT3Transformer{}
			verifier := NewVerifier(StaticKey([]byte("secret"))).
				AllowSHA1(true).
				ReferenceResolver(FSReferenceResolver(os.DirFS("testdata/interop"))).
				XSLTTransformer(verifyTransformer)
			result, err := verifier.Verify(t.Context(), doc)
			require.NoError(t, err)
			require.Len(t, result.References, 1)
		})
	}
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

package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"slices"
	"sync"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const referenceURIC1 = "#c1"

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

func TestVerifyPreflightsAllReferencesBeforeXSLT(t *testing.T) {
	tests := []struct {
		name          string
		allowXPointer bool
		wantErr       error
		mutateSecond  func(*testing.T, *helium.Document, *helium.Element)
	}{
		{
			name:    "XPath filter",
			wantErr: ErrUnsupportedTransform,
			mutateSecond: func(t *testing.T, doc *helium.Document, ref *helium.Element) {
				transform := findChild(t, findChild(t, ref, "Transforms"), "Transform")
				require.NoError(t, transform.SetAttribute("Algorithm", TransformXPath))
				xpathElem, err := doc.CreateElement("XPath")
				require.NoError(t, err)
				require.NoError(t, xpathElem.SetActiveNamespace(nsPrefix, NamespaceDSig))
				require.NoError(t, xpathElem.AddChild(doc.CreateText([]byte("$missing"))))
				require.NoError(t, transform.AddChild(xpathElem))
			},
		},
		{
			name:          "general XPointer",
			allowXPointer: true,
			wantErr:       ErrReferenceNotFound,
			mutateSecond: func(t *testing.T, _ *helium.Document, ref *helium.Element) {
				require.NoError(t, ref.SetAttribute("URI", "#xpointer(/doc[true() or $missing])"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			require.NoError(t, err)

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc><content Id="c1">payload</content></doc>`))
			require.NoError(t, err)
			ref := ReferenceConfig{
				URI:             referenceURIC1,
				DigestAlgorithm: DigestSHA256,
				Transforms:      []Transform{C14NTransform(C14N10)},
			}
			sigElem, err := NewSigner().
				SignatureAlgorithm(AlgRSASHA256).
				Reference(ref).
				Reference(ref).
				SignDetached(t.Context(), doc, key)
			require.NoError(t, err)
			require.NoError(t, doc.DocumentElement().AddChild(sigElem))

			signedInfo := findChild(t, sigElem, "SignedInfo")
			var refs []*helium.Element
			for child := signedInfo.FirstChild(); child != nil; child = child.NextSibling() {
				elem, ok := helium.AsNode[*helium.Element](child)
				if ok && elem.LocalName() == "Reference" {
					refs = append(refs, elem)
				}
			}
			require.Len(t, refs, 2)

			firstTransform := findChild(t, findChild(t, refs[0], "Transforms"), "Transform")
			require.NoError(t, firstTransform.SetAttribute("Algorithm", TransformXSLT))
			stylesheet, err := doc.CreateElement("stylesheet")
			require.NoError(t, err)
			require.NoError(t, stylesheet.DeclareNamespace("xsl", namespaceXSLT))
			require.NoError(t, stylesheet.SetActiveNamespace("xsl", namespaceXSLT))
			require.NoError(t, stylesheet.SetAttribute("version", "1.0"))
			require.NoError(t, firstTransform.AddChild(stylesheet))

			test.mutateSecond(t, doc, refs[1])
			reSignSignedInfo(t, doc, sigElem, signedInfo, nil, key)

			transformer := &pipelineRecordingTransformer{}
			verifier := NewVerifier(StaticKey(&key.PublicKey)).XSLTTransformer(transformer)
			if test.allowXPointer {
				verifier = verifier.AllowXPointer(true)
			}
			_, err = verifier.Verify(t.Context(), doc)
			require.ErrorIs(t, err, test.wantErr)
			var verifyErr *VerificationError
			require.ErrorAs(t, err, &verifyErr)
			require.Equal(t, 1, verifyErr.Reference)
			require.Empty(t, transformer.snapshot(), "an earlier XSLT transform must not run before all References pass static validation")
		})
	}
}

func TestVerifyPreflightsAllReferencesBeforeResolver(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc><content Id="c1">payload</content></doc>`))
	require.NoError(t, err)
	resolver := &countingResolver{}
	sigElem, err := NewSigner().
		SignatureAlgorithm(AlgRSASHA256).
		ReferenceResolver(resolver).
		Reference(ReferenceConfig{URI: "external.xml", DigestAlgorithm: DigestSHA256}).
		Reference(ReferenceConfig{
			URI:             referenceURIC1,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{C14NTransform(C14N10)},
		}).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sigElem))
	resolver.calls = 0

	signedInfo := findChild(t, sigElem, "SignedInfo")
	var refs []*helium.Element
	for child := signedInfo.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := helium.AsNode[*helium.Element](child)
		if ok && elem.LocalName() == "Reference" {
			refs = append(refs, elem)
		}
	}
	require.Len(t, refs, 2)
	transform := findChild(t, findChild(t, refs[1], "Transforms"), "Transform")
	require.NoError(t, transform.SetAttribute("Algorithm", TransformXPath))
	xpathElem, err := doc.CreateElement("XPath")
	require.NoError(t, err)
	require.NoError(t, xpathElem.SetActiveNamespace(nsPrefix, NamespaceDSig))
	require.NoError(t, xpathElem.AddChild(doc.CreateText([]byte("$missing"))))
	require.NoError(t, transform.AddChild(xpathElem))
	reSignSignedInfo(t, doc, sigElem, signedInfo, nil, key)

	_, err = NewVerifier(StaticKey(&key.PublicKey)).
		ReferenceResolver(resolver).
		Verify(t.Context(), doc)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	var verifyErr *VerificationError
	require.ErrorAs(t, err, &verifyErr)
	require.Equal(t, 1, verifyErr.Reference)
	require.Zero(t, resolver.calls, "an earlier Reference resolver must not run before all References pass static validation")
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

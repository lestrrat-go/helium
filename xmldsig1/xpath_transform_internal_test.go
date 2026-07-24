package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const referenceURIC1 = "#c1"

type countingXSLTTransformer struct {
	calls atomic.Int32
}

func (t *countingXSLTTransformer) TransformXSLT(_ context.Context, _, input []byte) ([]byte, error) {
	t.calls.Add(1)
	return input, nil
}

type countingReferenceResolver struct {
	calls  atomic.Int32
	octets []byte
}

func (r *countingReferenceResolver) ResolveReference(_ context.Context, _ string) ([]byte, error) {
	r.calls.Add(1)
	return r.octets, nil
}

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
	require.NotNil(t, pipe.xpathFilters[0].prepared, "pipeline resolution must prepare the filter for execution")
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

func TestXPathFilterValidatesEmptyNodeSet(t *testing.T) {
	filtered, err := applyXPathFilter(t.Context(), nil, xpathFilter{expr: "$missing"})
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	require.Nil(t, filtered)
}

func TestXPathTransformPreflightsAllFiltersBeforeXSLT(t *testing.T) {
	const reference = `<Reference xmlns="http://www.w3.org/2000/09/xmldsig#" URI="#c1">
		<Transforms>
			<Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116"><XPath>false()</XPath></Transform>
			<Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116"><XPath>$missing</XPath></Transform>
			<Transform Algorithm="http://www.w3.org/TR/1999/REC-xslt-19991116">
				<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">
					<xsl:template match="/"><out/></xsl:template>
				</xsl:stylesheet>
			</Transform>
		</Transforms>
		<DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
		<DigestValue>AA==</DigestValue>
	</Reference>`
	ref, err := parseReferenceFragment(t, reference)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc><content Id="c1">payload</content></doc>`))
	require.NoError(t, err)

	transformer := &countingXSLTTransformer{}
	_, octets, _, err := canonicalizeReference(t.Context(), &verifierConfig{xsltTransformer: transformer}, doc, nil, ref)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	require.Nil(t, octets)
	require.Zero(t, transformer.calls.Load(), "XSLT must not run after a statically invalid XPath filter")
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			tt.mutateSecond(t, doc, refs[1])
			reSignSignedInfo(t, doc, sigElem, signedInfo, nil, key)

			transformer := &countingXSLTTransformer{}
			verifier := NewVerifier(StaticKey(&key.PublicKey)).XSLTTransformer(transformer)
			if tt.allowXPointer {
				verifier = verifier.AllowXPointer(true)
			}
			_, err = verifier.Verify(t.Context(), doc)
			require.ErrorIs(t, err, tt.wantErr)
			var verifyErr *VerificationError
			require.ErrorAs(t, err, &verifyErr)
			require.Equal(t, 1, verifyErr.Reference)
			require.Zero(t, transformer.calls.Load(), "an earlier XSLT transform must not run before all References pass static validation")
		})
	}
}

func TestVerifyPreflightsAllReferencesBeforeResolver(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc><content Id="c1">payload</content></doc>`))
	require.NoError(t, err)
	resolver := &countingReferenceResolver{octets: []byte("external payload")}
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
	resolver.calls.Store(0)

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
	require.Zero(t, resolver.calls.Load(), "an earlier Reference resolver must not run before all References pass static validation")
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

	// defCan-2/3 carry an XSLT transform. The stylesheet subtree now parses (the
	// XSLT transform is captured, not rejected at parse), but both vectors order
	// the XSLT transform AFTER a C14N11 that already produced octets, so resolving
	// the transform pipeline rejects the mis-ordered chain fail-closed — regardless
	// of whether an XSLTTransformer is configured.
	t.Run("defCan-2 XSLT-after-c14n chain rejected at resolve", func(t *testing.T) {
		_, parsed := parseVectorSignature(t, "defCan-2-signature.xml")
		require.Len(t, parsed.references, 1)
		_, err := resolveTransformPipeline(stepsFromParsed(parsed.references[0]))
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("defCan-3 XSLT-after-c14n chain rejected at resolve", func(t *testing.T) {
		_, parsed := parseVectorSignature(t, "defCan-3-signature.xml")
		require.Len(t, parsed.references, 1)
		_, err := resolveTransformPipeline(stepsFromParsed(parsed.references[0]))
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

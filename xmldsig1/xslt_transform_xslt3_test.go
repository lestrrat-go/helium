package xmldsig1

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// xslt3Transformer is a test-only XSLTTransformer backed by the helium xslt3
// engine. Production xmldsig1 code never imports xslt3 (it is the security layer,
// with a deliberately minimal import set); the dependency lives ONLY in this test
// file, exercising the injected-transformer seam exactly as a real embedder would.
//
// It performs NO resource limiting or XXE hardening beyond xslt3's own hardened
// defaults, which is acceptable for a fixed, trusted test stylesheet. A real
// deployment verifying attacker-controlled signatures owns those limits (see the
// XSLTTransformer doc comment).
type xslt3Transformer struct{}

func (xslt3Transformer) TransformXSLT(ctx context.Context, stylesheet, input []byte) ([]byte, error) {
	ssDoc, err := helium.NewParser().Parse(ctx, stylesheet)
	if err != nil {
		return nil, err
	}
	ss, err := xslt3.CompileStylesheet(ctx, ssDoc)
	if err != nil {
		return nil, err
	}
	srcDoc, err := helium.NewParser().Parse(ctx, input)
	if err != nil {
		return nil, err
	}
	out, err := xslt3.TransformString(ctx, srcDoc, ss)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// TestXSLTTransformDigestOverStylesheetOutput drives a same-document Reference
// carrying an XSLT transform through canonicalizeReference with a real
// xslt3-backed transformer, and proves the digest input is the STYLESHEET OUTPUT,
// not the pre-XSLT canonical octets. It parses the ds:Transform/xsl:stylesheet
// subtree through the real parse path, so the transformer receives exactly the
// bytes the verifier captures.
func TestXSLTTransformDigestOverStylesheetOutput(t *testing.T) {
	// A same-document reference to <content Id="c1">. The stylesheet rewrites the
	// input into a <wrapped> element carrying the greeting text, so its output is
	// distinct from the pre-XSLT canonical octets of the content subtree.
	const doc = `<doc>` +
		`<content Id="c1"><greeting>hello</greeting></content>` +
		`<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		`<ds:SignedInfo>` +
		`<ds:CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"/>` +
		`<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#hmac-sha256"/>` +
		`<ds:Reference URI="#c1">` +
		`<ds:Transforms><ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xslt-19991116">` +
		`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">` +
		`<xsl:output method="xml" omit-xml-declaration="yes"/>` +
		`<xsl:template match="/"><wrapped><xsl:value-of select="//greeting"/></wrapped></xsl:template>` +
		`</xsl:stylesheet>` +
		`</ds:Transform></ds:Transforms>` +
		`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>` +
		`<ds:DigestValue>AA==</ds:DigestValue>` +
		`</ds:Reference>` +
		`</ds:SignedInfo>` +
		`</ds:Signature>` +
		`</doc>`

	parsedDoc, err := helium.NewParser().Parse(t.Context(), []byte(doc))
	require.NoError(t, err)

	refElem := findLocal(parsedDoc.DocumentElement(), "Reference")
	require.NotNil(t, refElem)
	ref, err := parseReferenceElement(t.Context(), testVerifyBudget(), refElem)
	require.NoError(t, err)
	require.Len(t, ref.transforms, 1)
	require.Equal(t, TransformXSLT, ref.transforms[0].algorithm)

	cfg := &verifierConfig{xsltTransformer: xslt3Transformer{}}
	target, octets, external, err := canonicalizeReference(t.Context(), cfg, parsedDoc, nil, ref)
	require.NoError(t, err)
	require.False(t, external)
	require.Equal(t, "content", target.LocalName())

	// The digest input must equal the transformer's output over the pre-XSLT
	// octets (the C14N 1.0 canonicalization of the content subtree), NOT the
	// pre-XSLT octets themselves.
	preOctets, err := canonicalizeSubtree(C14N10, target, nil)
	require.NoError(t, err)
	expected, err := xslt3Transformer{}.TransformXSLT(t.Context(), ref.transforms[0].stylesheet, preOctets)
	require.NoError(t, err)

	require.Equal(t, expected, octets, "the digest input must be the stylesheet output")
	require.NotEqual(t, preOctets, octets, "the digest input must NOT be the pre-XSLT octets")
	require.Contains(t, string(octets), "wrapped", "the stylesheet output shape must survive to the digest input")
	require.Contains(t, string(octets), "hello")

	// And the recomputed digest over that output matches what a DigestValue built
	// from the stylesheet output would carry.
	digest, err := computeDigest(ref.digestAlgorithm, octets, false)
	require.NoError(t, err)
	require.NotEmpty(t, digest)
}

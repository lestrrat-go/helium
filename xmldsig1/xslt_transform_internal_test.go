package xmldsig1

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// parseReferenceFragment parses a standalone ds:Reference XML fragment and runs
// parseReferenceElement over it, mirroring how parseSignedInfo reaches a
// Reference during verification.
func parseReferenceFragment(t *testing.T, frag string) (parsedReference, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(frag))
	require.NoError(t, err)
	ref := findLocal(doc.DocumentElement(), "Reference")
	require.NotNil(t, ref, "fragment must contain a ds:Reference")
	return parseReferenceElement(context.Background(), testVerifyBudget(), ref)
}

// TestParseXSLTTransform locks the parse-side handling of the XSLT transform: a
// valid ds:Transform/xsl:stylesheet subtree is captured, and a missing,
// duplicate, or foreign stylesheet child is rejected fail-closed.
func TestParseXSLTTransform(t *testing.T) {
	const head = `<Reference xmlns="http://www.w3.org/2000/09/xmldsig#" URI="#x"><Transforms><Transform Algorithm="http://www.w3.org/TR/1999/REC-xslt-19991116">`
	const tail = `</Transform></Transforms><DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><DigestValue>AA==</DigestValue></Reference>`

	t.Run("valid stylesheet captured", func(t *testing.T) {
		ref, err := parseReferenceFragment(t, head+
			`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"><xsl:template match="/"><out/></xsl:template></xsl:stylesheet>`+
			tail)
		require.NoError(t, err)
		require.Len(t, ref.transforms, 1)
		require.Equal(t, TransformXSLT, ref.transforms[0].algorithm)
		require.NotEmpty(t, ref.transforms[0].stylesheet, "the stylesheet subtree must be captured")
		// The captured bytes are the canonicalized stylesheet subtree, so they carry
		// the xsl: namespace declaration and re-parse as a standalone stylesheet.
		require.Contains(t, string(ref.transforms[0].stylesheet), "stylesheet")
		require.Contains(t, string(ref.transforms[0].stylesheet), `xmlns:xsl="http://www.w3.org/1999/XSL/Transform"`)
	})

	t.Run("xsl:transform synonym captured", func(t *testing.T) {
		ref, err := parseReferenceFragment(t, head+
			`<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"><xsl:template match="/"><out/></xsl:template></xsl:transform>`+
			tail)
		require.NoError(t, err)
		require.Len(t, ref.transforms, 1)
		require.NotEmpty(t, ref.transforms[0].stylesheet)
	})

	t.Run("missing stylesheet child rejected", func(t *testing.T) {
		_, err := parseReferenceFragment(t, head+tail)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("duplicate stylesheet child rejected", func(t *testing.T) {
		_, err := parseReferenceFragment(t, head+
			`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/>`+
			`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/>`+
			tail)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("foreign child rejected", func(t *testing.T) {
		// A same-local-name element in a foreign namespace must not be accepted as
		// the stylesheet root.
		_, err := parseReferenceFragment(t, head+
			`<stylesheet xmlns="urn:not-xslt"/>`+
			tail)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})
}

// TestParseXSLTTransformEntityHidden locks the fail-closed contract against an
// entity-reference child that hides extra markup. helium.NewParser() is
// non-substituting by default, so an internal DTD entity referenced inside the
// ds:Transform stays an EntityRefNode instead of being inlined. An element-only
// scan would miss the hidden second/foreign stylesheet and wrongly accept the
// transform; parseXSLTTransform must reject any entity-reference child so exactly
// one direct xsl:stylesheet/xsl:transform element is permitted.
func TestParseXSLTTransformEntityHidden(t *testing.T) {
	const head = `<Reference xmlns="http://www.w3.org/2000/09/xmldsig#" URI="#x"><Transforms><Transform Algorithm="http://www.w3.org/TR/1999/REC-xslt-19991116">`
	const tail = `</Transform></Transforms><DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><DigestValue>AA==</DigestValue></Reference>`
	const style = `<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/>`

	t.Run("entity replacement is a second stylesheet rejected", func(t *testing.T) {
		// Children parse as [Element xsl:stylesheet, EntityRefNode evil]; the
		// entity replacement is a second stylesheet the element-only scan cannot
		// see. Must fail closed rather than accept the single visible element.
		doctype := `<!DOCTYPE Reference [ <!ENTITY evil "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='1.0'/>"> ]>`
		_, err := parseReferenceFragment(t, doctype+head+style+`&evil;`+tail)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("entity replacement is a foreign element rejected", func(t *testing.T) {
		doctype := `<!DOCTYPE Reference [ <!ENTITY foreign "<inject xmlns='urn:not-xslt'/>"> ]>`
		_, err := parseReferenceFragment(t, doctype+head+style+`&foreign;`+tail)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("legitimate single stylesheet still parses", func(t *testing.T) {
		// A ds:Transform with exactly one xsl:stylesheet and no entity child is
		// unaffected by the entity guard.
		ref, err := parseReferenceFragment(t, head+style+tail)
		require.NoError(t, err)
		require.Len(t, ref.transforms, 1)
		require.Equal(t, TransformXSLT, ref.transforms[0].algorithm)
		require.NotEmpty(t, ref.transforms[0].stylesheet)
	})
}

// TestResolveXSLTPipeline locks the pipeline placement of the XSLT transform: a
// bare XSLT resolves with the default inclusive C14N 1.0 fill-in and pipe.xslt
// set, a preceding node-set transform is allowed, and any octet-ender before or
// after the XSLT (or a second XSLT) is rejected fail-closed.
func TestResolveXSLTPipeline(t *testing.T) {
	xslt := transformStep{algorithm: TransformXSLT, stylesheet: []byte("<xsl/>")}

	t.Run("bare XSLT resolves with c14n fill-in", func(t *testing.T) {
		pipe, err := resolveTransformPipeline([]transformStep{xslt})
		require.NoError(t, err)
		require.NotNil(t, pipe.xslt, "pipe.xslt must mark the XSLT transform")
		require.Equal(t, C14N10, pipe.c14nMethod, "a bare XSLT uses the inclusive C14N 1.0 fill-in for its pre-XSLT octets")
	})

	t.Run("node-set transform may precede XSLT", func(t *testing.T) {
		pipe, err := resolveTransformPipeline([]transformStep{
			{algorithm: TransformEnvelopedSignature},
			{algorithm: TransformXPath, xpathExpr: "true()"},
			xslt,
		})
		require.NoError(t, err)
		require.NotNil(t, pipe.xslt)
		require.True(t, pipe.hasEnveloped)
		require.Len(t, pipe.xpathFilters, 1)
	})

	t.Run("XSLT after XSLT rejected", func(t *testing.T) {
		_, err := resolveTransformPipeline([]transformStep{xslt, xslt})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("XSLT after c14n rejected", func(t *testing.T) {
		_, err := resolveTransformPipeline([]transformStep{{algorithm: C14N11URI}, xslt})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("c14n after XSLT rejected", func(t *testing.T) {
		_, err := resolveTransformPipeline([]transformStep{xslt, {algorithm: C14N11URI}})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("XSLT with base64 rejected either order", func(t *testing.T) {
		_, err := resolveTransformPipeline([]transformStep{{algorithm: TransformBase64}, xslt})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		_, err = resolveTransformPipeline([]transformStep{xslt, {algorithm: TransformBase64}})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})
}

// TestXSLTFailClosedWithoutTransformer confirms a Reference carrying an XSLT
// transform fails closed with ErrUnsupportedTransform when no XSLTTransformer is
// configured — helium never runs attacker-controlled XSLT on its own.
func TestXSLTFailClosedWithoutTransformer(t *testing.T) {
	const xml = `<doc><content Id="c1"><g>hi</g></content></doc>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	ref := parsedReference{
		uri:             "#c1",
		digestAlgorithm: DigestSHA256,
		transforms: []parsedTransform{
			{algorithm: TransformXSLT, stylesheet: []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/>`)},
		},
	}

	_, octets, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	require.Nil(t, octets)
}

// TestXSLTSignPreflightRejected confirms the sign preflight rejects an XSLT
// transform fail-closed. Signing has no typed XSLT constructor, so a caller must
// implement the exported Transform interface with the XSLT URI; the digest path
// has no XSLT branch, so honoring it would fail open.
func TestXSLTSignPreflightRejected(t *testing.T) {
	cfg := &signerConfig{
		references: []ReferenceConfig{
			{URI: "#c1", DigestAlgorithm: DigestSHA256, Transforms: []Transform{customTransform{uri: TransformXSLT}}},
		},
	}
	err := preflightSignerTransforms(cfg)
	require.ErrorIs(t, err, ErrUnsupportedTransform)

	var refErr *ReferenceError
	require.ErrorAs(t, err, &refErr)
	require.Equal(t, opSign, refErr.Op)
	require.Equal(t, 0, refErr.Reference)
}

// customTransform is a caller-supplied Transform with an arbitrary URI, used to
// prove the sign preflight rejects an XSLT transform even though the package
// ships no typed XSLT constructor.
type customTransform struct {
	uri string
}

func (c customTransform) URI() string { return c.uri }

// markerXSLTTransformer is a POINTER-receiver XSLTTransformer whose method
// dereferences its receiver (reading p.marker). A typed-nil
// (*markerXSLTTransformer)(nil) yields a non-nil interface wrapping a nil
// pointer, so a plain cfg.xsltTransformer == nil check misses it and the call
// would panic on the nil-receiver dereference. The XSLT gate must instead treat
// a typed-nil value as absent and fail closed with ErrUnsupportedTransform. When
// non-nil it prefixes its marker to the input so a test can prove the
// transformer's output — not the pre-XSLT octets — became the digest input.
type markerXSLTTransformer struct{ marker string }

func (p *markerXSLTTransformer) TransformXSLT(_ context.Context, _, input []byte) ([]byte, error) {
	return append([]byte(p.marker), input...), nil
}

// TestXSLTTypedNilTransformerFailsClosed covers the same-document XSLT gate: a
// typed-nil pointer XSLTTransformer must be treated as absent and fail closed
// with ErrUnsupportedTransform rather than panic on the nil-receiver call.
func TestXSLTTypedNilTransformerFailsClosed(t *testing.T) {
	const xml = `<doc><content Id="c1"><g>hi</g></content></doc>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	ref := parsedReference{
		uri:             "#c1",
		digestAlgorithm: DigestSHA256,
		transforms: []parsedTransform{
			{algorithm: TransformXSLT, stylesheet: []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/>`)},
		},
	}

	var tr *markerXSLTTransformer // typed-nil pointer: non-nil interface, nil value
	cfg := &verifierConfig{xsltTransformer: tr}

	require.NotPanics(t, func() {
		_, octets, _, err := canonicalizeReference(t.Context(), cfg, doc, nil, ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Nil(t, octets)
	})
}

// TestExternalXSLTTransform covers the external-reference XSLT path: a bare XSLT
// transform on an external Reference must fail closed when no (or a typed-nil)
// transformer is configured, and must digest the CONFIGURED transformer's output
// when one is present — never the untransformed resolved octets.
func TestExternalXSLTTransform(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	octets := []byte(`<ext>payload</ext>`)
	res := staticResolver(octets)

	xsltRef := parsedReference{
		uri: "http://example.com/x",
		transforms: []parsedTransform{
			{algorithm: TransformXSLT, stylesheet: []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/>`)},
		},
	}

	t.Run("absent transformer fails closed", func(t *testing.T) {
		cfg := &verifierConfig{referenceResolver: res}
		out, err := resolveExternalReference(t.Context(), cfg, doc, xsltRef)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Nil(t, out)
	})

	t.Run("typed-nil transformer fails closed", func(t *testing.T) {
		var tr *markerXSLTTransformer
		cfg := &verifierConfig{referenceResolver: res, xsltTransformer: tr}
		require.NotPanics(t, func() {
			out, err := resolveExternalReference(t.Context(), cfg, doc, xsltRef)
			require.ErrorIs(t, err, ErrUnsupportedTransform)
			require.Nil(t, out)
		})
	})

	t.Run("configured transformer output is digested", func(t *testing.T) {
		tr := &markerXSLTTransformer{marker: "XSLT:"}
		cfg := &verifierConfig{referenceResolver: res, xsltTransformer: tr}
		out, err := resolveExternalReference(t.Context(), cfg, doc, xsltRef)
		require.NoError(t, err)
		// A bare XSLT external reference hands the raw resolved octets to the
		// transformer as its pre-XSLT input; the transformer's output (marker +
		// octets) is the digest input, proving the XSLT transform ran.
		require.Equal(t, append([]byte("XSLT:"), octets...), out)
	})
}

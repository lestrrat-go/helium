package xslt3_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// innerTransformStylesheet is a small stylesheet exercised through the
// standalone fn:transform (xslt3.TransformFunction) registered on a bare
// xpath3.Evaluator. The match="/" template echoes the source root element name;
// the named template emits a fixed marker.
const innerTransformStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="name(*)"/></out></xsl:template>
  <xsl:template name="go"><out>named-template</out></xsl:template>
</xsl:stylesheet>`

// evalTransform compiles and evaluates expr against sourceDoc as the context
// node, with the supplied variable bindings and fn:transform behavior (found or
// stub) determined by fns.
func evalTransform(t *testing.T, expr string, sourceDoc *helium.Document, vars map[string]xpath3.Sequence, fns map[xpath3.QualifiedName]xpath3.Function) (string, error) {
	t.Helper()
	e := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	if vars != nil {
		e = e.Variables(vars)
	}
	if fns != nil {
		e = e.Functions(nil, fns)
	}
	x, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	res, err := e.Evaluate(t.Context(), x, sourceDoc)
	if err != nil {
		return "", err
	}
	return res.StringValue(), nil
}

func transformFns() map[xpath3.QualifiedName]xpath3.Function {
	return map[xpath3.QualifiedName]xpath3.Function{
		{URI: xpath3.NSFn, Name: "transform"}: xslt3.TransformFunction(),
	}
}

// mapURIResolver serves a fixed set of URIs from an in-memory map, used to
// exercise relative xsl:include resolution in the fn:transform
// stylesheet-base-uri tests.
type mapURIResolver struct {
	files map[string]string
}

func (r mapURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	content, ok := r.files[uri]
	if !ok {
		return nil, fmt.Errorf("mapURIResolver: no such URI %q", uri)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// recordingRejectResolver records every URI it is asked for and refuses to
// serve any of them. It lets a test prove which URI the compiler attempted to
// load (e.g. a raw relative href when no base URI was applied) rather than only
// observing that resolution failed.
type recordingRejectResolver struct {
	requested []string
}

func (r *recordingRejectResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	r.requested = append(r.requested, uri)
	return nil, fmt.Errorf("recordingRejectResolver: refusing %q", uri)
}

// includedTemplateStylesheet is served as an xsl:include target; its match="/"
// template echoes the source root element name.
const includedTemplateStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="name(*)"/></out></xsl:template>
</xsl:stylesheet>`

// includingStylesheet pulls in the template above via a relative href, so it
// only compiles when a usable base URI resolves the include.
const includingStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="sub/inc.xsl"/>
</xsl:stylesheet>`

// TestTransformFunctionStylesheetBaseURI verifies that fn:transform honors the
// stylesheet-base-uri option (and a stylesheet-node's own document base URI)
// when resolving a relative xsl:include inside a stylesheet-text/-node.
func TestTransformFunctionStylesheetBaseURI(t *testing.T) {
	sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<data>hi</data>`))
	require.NoError(t, err)

	// The include href "sub/inc.xsl" resolves against the base
	// "http://example.com/base/main.xsl" to this URI.
	resolver := mapURIResolver{files: map[string]string{
		"http://example.com/base/sub/inc.xsl": includedTemplateStylesheet,
	}}
	fnsWith := func(opts ...xslt3.TransformOption) map[xpath3.QualifiedName]xpath3.Function {
		return map[xpath3.QualifiedName]xpath3.Function{
			{URI: xpath3.NSFn, Name: "transform"}: xslt3.TransformFunction(
				append([]xslt3.TransformOption{xslt3.WithTransformURIResolver(resolver)}, opts...)...),
		}
	}

	// With no stylesheet-base-uri and no call static base URI, the relative
	// include href gets no base applied, so the compiler attempts to load the
	// RAW relative "sub/inc.xsl". A recording resolver proves that unbased URI is
	// what is attempted (not a resolver-file-absence for some based URI) and
	// that the transform then fails — the genuine no-usable-base path that
	// fn-transform-err-9 exercises. (helium reports XTSE0165 when NO resolver at
	// all is configured; here a resolver is present but the unbased relative URI
	// is unresolvable regardless.)
	t.Run("StylesheetTextNoBaseAttemptsRawRelative", func(t *testing.T) {
		rec := &recordingRejectResolver{}
		fns := map[xpath3.QualifiedName]xpath3.Function{
			{URI: xpath3.NSFn, Name: "transform"}: xslt3.TransformFunction(xslt3.WithTransformURIResolver(rec)),
		}
		_, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(includingStylesheet)},
			fns,
		)
		require.Error(t, err)
		// The unbased relative href is what the compiler tried to load — never an
		// absolute/based URI. This distinguishes the no-base path from mere
		// resolver-file-absence.
		require.Equal(t, []string{"sub/inc.xsl"}, rec.requested)
	})

	// With no resolver at all, the relative include cannot be loaded and the
	// nested compile fails with the specific XTSE0165 (opt-in resolver) error.
	t.Run("StylesheetTextNoResolverXTSE0165", func(t *testing.T) {
		_, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(includingStylesheet)},
			transformFns(),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "XTSE0165")
	})

	// stylesheet-base-uri (absolute) supplies the base for the inline text so the
	// relative include resolves.
	t.Run("StylesheetTextAbsoluteBaseURI", func(t *testing.T) {
		out, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-base-uri': $base, 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{
				"ss":   xpath3.SingleString(includingStylesheet),
				"base": xpath3.SingleString("http://example.com/base/main.xsl"),
			},
			fnsWith(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "<out>data</out>")
	})

	// A relative stylesheet-base-uri is resolved against the call's static base
	// URI (WithTransformBaseURI) before it is used as the include base.
	t.Run("StylesheetTextRelativeBaseURI", func(t *testing.T) {
		out, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-base-uri': $base, 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{
				"ss":   xpath3.SingleString(includingStylesheet),
				"base": xpath3.SingleString("base/main.xsl"),
			},
			fnsWith(xslt3.WithTransformBaseURI("http://example.com/root.xsl")),
		)
		require.NoError(t, err)
		require.Contains(t, out, "<out>data</out>")
	})

	// stylesheet-node defaults its base URI from the node's own document base
	// URI (fn-transform-24 semantics): a relative include resolves without any
	// stylesheet-base-uri option.
	t.Run("StylesheetNodeDocBaseDefault", func(t *testing.T) {
		ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(includingStylesheet))
		require.NoError(t, err)
		ssDoc.SetURL("http://example.com/base/main.xsl")
		out, err := evalTransform(t,
			`transform(map{'stylesheet-node': $ssnode, 'source-node': ., 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ssnode": xpath3.ItemSlice{xpath3.NodeItem{Node: ssDoc}}},
			fnsWith(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "<out>data</out>")
	})

	// stylesheet-base-uri overrides the stylesheet-node's own document base URI
	// (fn-transform-23 semantics).
	t.Run("StylesheetNodeBaseURIOption", func(t *testing.T) {
		ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(includingStylesheet))
		require.NoError(t, err)
		ssDoc.SetURL("http://elsewhere.example.org/wrong/main.xsl")
		out, err := evalTransform(t,
			`transform(map{'stylesheet-node': $ssnode, 'source-node': ., 'stylesheet-base-uri': $base, 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{
				"ssnode": xpath3.ItemSlice{xpath3.NodeItem{Node: ssDoc}},
				"base":   xpath3.SingleString("http://example.com/base/main.xsl"),
			},
			fnsWith(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "<out>data</out>")
	})
}

// TestTransformFunctionStandalone drives xslt3.TransformFunction as a
// registered xpath3 function through a real xpath3.Evaluator — the standalone
// path that the QT3 harness exercises (no outer running stylesheet).
func TestTransformFunctionStandalone(t *testing.T) {
	sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<data>hi</data>`))
	require.NoError(t, err)

	// Baseline: the bare xpath3 stub (no injected fn:transform) is
	// unimplemented, so the standalone path needs the xslt3 injection.
	t.Run("StubUnimplemented", func(t *testing.T) {
		_, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
			nil,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not implemented")
	})

	t.Run("StylesheetText", func(t *testing.T) {
		out, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "<out>data</out>")
	})

	t.Run("StylesheetNode", func(t *testing.T) {
		ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(innerTransformStylesheet))
		require.NoError(t, err)
		out, err := evalTransform(t,
			`transform(map{'stylesheet-node': $ssnode, 'source-node': ., 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ssnode": xpath3.ItemSlice{xpath3.NodeItem{Node: ssDoc}}},
			transformFns(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "<out>data</out>")
	})

	t.Run("InitialTemplate", func(t *testing.T) {
		out, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'initial-template': QName('','go'), 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "<out>named-template</out>")
	})

	t.Run("DocumentDelivery", func(t *testing.T) {
		out, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'document'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "data")
	})
}

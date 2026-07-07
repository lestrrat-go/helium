package xslt3_test

import (
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

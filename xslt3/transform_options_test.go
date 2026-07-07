package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// fn:transform error codes asserted by the option-validation table.
const (
	wantFOXT0002 = "FOXT0002"
	wantXPTY0004 = "XPTY0004"
)

// XSLT-namespace requested-property local names asserted by the capability table.
const (
	propSupportsStreaming = "supports-streaming"
	propIsSchemaAware     = "is-schema-aware"
)

// nsInitialTemplateStylesheet declares a named template in a non-empty
// namespace (app:main). It exercises the initial-template QName resolution: a
// QName-valued initial-template option must keep its namespace so the
// {http://www.example.com}main template is found (QT3 fn-transform-10/11).
const nsInitialTemplateStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:app="http://www.example.com">
  <xsl:template match="/"><x><xsl:value-of select="."/></x></xsl:template>
  <xsl:template name="app:main"><out>ns-named</out></xsl:template>
</xsl:stylesheet>`

// TestTransformInitialTemplateQName proves sub-fix 1: an initial-template
// supplied as a namespaced xs:QName resolves to the namespaced named template
// rather than dropping the namespace and failing with XTDE0820.
func TestTransformInitialTemplateQName(t *testing.T) {
	sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>this</doc>`))
	require.NoError(t, err)

	t.Run("NamespacedQNameResolves", func(t *testing.T) {
		out, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-template': QName('http://www.example.com','main'), 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(nsInitialTemplateStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)
		require.Contains(t, out, "ns-named")
		require.Contains(t, out, "<out")
	})

	// A namespaced QName whose namespace does not match any template still fails,
	// confirming the namespace participates in the lookup (not silently dropped).
	t.Run("WrongNamespaceFails", func(t *testing.T) {
		_, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-template': QName('http://wrong.example.org','main'), 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(nsInitialTemplateStylesheet)},
			transformFns(),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "XTDE0820")
	})
}

// simpleInvokableStylesheet has both a match="/" rule and a named template so it
// is invokable through any entry point the validation tests supply.
const simpleInvokableStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="."/></out></xsl:template>
  <xsl:template name="t"><out>named</out></xsl:template>
</xsl:stylesheet>`

// TestTransformOptionValidation proves sub-fix 2: fn:transform rejects invalid,
// mutually-exclusive, and mistyped option combinations (QT3 fn-transform-err-2,
// -3, -4, -5, -6, -13, -17, -18).
func TestTransformOptionValidation(t *testing.T) {
	sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>this</doc>`))
	require.NoError(t, err)

	testCases := []struct {
		name string
		expr string
		code string
	}{
		{
			// err-2: stylesheet-text and stylesheet-location are mutually exclusive.
			name: "TwoStylesheetSources_TextAndLocation",
			expr: `transform(map{'stylesheet-text': $ss, 'stylesheet-location': 'x.xsl', 'source-node': .})?output`,
			code: wantFOXT0002,
		},
		{
			// err-3: stylesheet-text and stylesheet-node are mutually exclusive.
			name: "TwoStylesheetSources_TextAndNode",
			expr: `transform(map{'stylesheet-text': $ss, 'stylesheet-node': ., 'source-node': .})?output`,
			code: wantFOXT0002,
		},
		{
			// err-6: initial-mode and initial-template are mutually exclusive.
			name: "TwoEntryPoints_ModeAndTemplate",
			expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-mode': QName('','m'), 'initial-template': QName('','t')})?output`,
			code: wantFOXT0002,
		},
		{
			// err-17: source-node and initial-match-selection are mutually exclusive.
			name: "SourceNodeAndInitialMatchSelection",
			expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-match-selection': .})?output`,
			code: wantFOXT0002,
		},
		{
			// err-13: delivery-format value not one of document/serialized/raw.
			name: "BadDeliveryFormat",
			expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'doc'})?output`,
			code: wantFOXT0002,
		},
		{
			// err-18: stylesheet-params keys must be QNames, not strings.
			name: "StylesheetParamsStringKey",
			expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-params': map{'debug': true()}})?output`,
			code: wantFOXT0002,
		},
		{
			// err-4: xslt-version value is mistyped (string, not numeric).
			name: "XSLTVersionWrongType",
			expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'xslt-version': '2.0'})?output`,
			code: wantXPTY0004,
		},
		{
			// err-5: stylesheet-params value is mistyped (string, not a map).
			name: "StylesheetParamsWrongType",
			expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-params': 'v'})?output`,
			code: wantXPTY0004,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := evalTransform(t, tc.expr, sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(simpleInvokableStylesheet)},
				transformFns(),
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.code)
		})
	}
}

// TestTransformRequestedProperties proves sub-fix 3: an unsatisfiable
// requested-properties entry raises FOXT0006. helium advertises
// supports-streaming = false and is otherwise schema-aware / backwards-compatible
// / namespace-axis-capable (QT3 fn-transform-71, -73, -75, -77).
func TestTransformRequestedProperties(t *testing.T) {
	sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>this</doc>`))
	require.NoError(t, err)

	const xsltNS = "http://www.w3.org/1999/XSL/Transform"

	unsatisfiable := []struct {
		name     string
		property string
		value    string
	}{
		// fn-transform-71: request a non-schema-aware processor.
		{"NonSchemaAware", propIsSchemaAware, "false()"},
		// fn-transform-73: request no backwards-compatibility support.
		{"NoBackwardsCompatibility", "supports-backwards-compatibility", "false()"},
		// fn-transform-75: request no namespace-axis support.
		{"NoNamespaceAxis", "supports-namespace-axis", "false()"},
		// fn-transform-77: request streaming support (helium does not stream).
		{"Streaming", propSupportsStreaming, "true()"},
		// A boolean requested as the xs:string lexical form "yes" is honored the
		// same as true() — an unsatisfiable streaming request still raises.
		{"StreamingYesString", propSupportsStreaming, `'yes'`},
		// The "1" lexical form is likewise a true value.
		{"StreamingOneString", propSupportsStreaming, `'1'`},
		// A false-valued property requested as "no" against an advertised-true
		// capability is also unsatisfiable.
		{"NoSchemaAwareNoString", propIsSchemaAware, `'no'`},
	}

	for _, tc := range unsatisfiable {
		t.Run("Unsatisfiable_"+tc.name, func(t *testing.T) {
			expr := `transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'requested-properties': map{QName('` + xsltNS + `','` + tc.property + `'): ` + tc.value + `}})?output`
			_, err := evalTransform(t, expr, sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(simpleInvokableStylesheet)},
				transformFns(),
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "FOXT0006")
		})
	}

	// Satisfiable requests must NOT raise: they match helium's advertised
	// capabilities, so the transform proceeds normally.
	satisfiable := []struct {
		name     string
		property string
		value    string
	}{
		{"SchemaAware", propIsSchemaAware, "true()"},
		{"BackwardsCompatibility", "supports-backwards-compatibility", "true()"},
		{"NamespaceAxis", "supports-namespace-axis", "true()"},
		{"NoStreaming", propSupportsStreaming, "false()"},
		// The xs:string lexical forms of a satisfiable request must also succeed.
		{"NoStreamingNoString", propSupportsStreaming, `'no'`},
		{"NoStreamingZeroString", propSupportsStreaming, `'0'`},
		{"SchemaAwareYesString", propIsSchemaAware, `'yes'`},
	}

	for _, tc := range satisfiable {
		t.Run("Satisfiable_"+tc.name, func(t *testing.T) {
			expr := `transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'requested-properties': map{QName('` + xsltNS + `','` + tc.property + `'): ` + tc.value + `}})?output`
			out, err := evalTransform(t, expr, sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(simpleInvokableStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>this</out>")
		})
	}
}

// nestedOutputStylesheet emits a nested element structure so that the indent
// serialization parameter produces observably different output.
const nestedOutputStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><a><b>x</b></a></out></xsl:template>
</xsl:stylesheet>`

// methodTextOutputStylesheet declares xsl:output method="text" so the inherited
// serialization method differs from the "xml" default. It lets a
// serialization-params entry present with an empty-sequence value be observed
// resetting the method back to its default.
const methodTextOutputStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/"><out><a><b>x</b></a></out></xsl:template>
</xsl:stylesheet>`

// TestTransformSerializationParams proves sub-fix 4: the serialization-params
// option is applied to the serialized delivery output (QT3 fn-transform-30/31).
func TestTransformSerializationParams(t *testing.T) {
	sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	// fn-transform-30: indent yes vs no must produce different serializations.
	t.Run("IndentChangesOutput", func(t *testing.T) {
		indented, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'indent': true()}})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(nestedOutputStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)

		compact, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'indent': false()}})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(nestedOutputStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)

		require.NotEqual(t, indented, compact, "indent=yes and indent=no must serialize differently")
	})

	// The method serialization parameter switches the output method: with
	// method=text only text nodes are emitted, so the element markup disappears.
	t.Run("MethodText", func(t *testing.T) {
		out, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'method': 'text'}})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(nestedOutputStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)
		require.Equal(t, "x", strings.TrimSpace(out))
	})

	// A serialization-params entry present with an empty-sequence value resets
	// that parameter to its serialization default, overriding the inherited
	// xsl:output value. Here method="text" from the stylesheet is reset to the
	// "xml" default, so element markup reappears in the output.
	t.Run("EmptySequenceResetsToDefault", func(t *testing.T) {
		// Baseline: inherited method="text" emits only the text node.
		inherited, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(methodTextOutputStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)
		require.Equal(t, "x", strings.TrimSpace(inherited))
		require.NotContains(t, inherited, "<out")

		// method: () resets the method to its "xml" default instead of leaving
		// the inherited text method in place, so the element markup is emitted.
		reset, err := evalTransform(t,
			`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'method': ()}})?output`,
			sourceDoc,
			map[string]xpath3.Sequence{"ss": xpath3.SingleString(methodTextOutputStylesheet)},
			transformFns(),
		)
		require.NoError(t, err)
		require.Contains(t, reset, "<out")
		require.Contains(t, reset, "<b>x</b>")
	})
}

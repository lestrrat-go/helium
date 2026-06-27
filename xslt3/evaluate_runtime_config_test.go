package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// evaluateXMLBaseStylesheet places an xml:base on the matching template so the
// template's static base URI differs from the stylesheet's base URI. A relative
// href passed to fn:unparsed-text must resolve against the TEMPLATE base, not
// the main stylesheet base. The %EXPR% placeholder runs the same retrieval
// statically (baseline) or through xsl:evaluate (dynamic).
const evaluateXMLBaseStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="u" select="'data.txt'"/>
  <xsl:template match="/" xml:base="sub/">
    <out>%EXPR%</out>
  </xsl:template>
</xsl:stylesheet>`

// TestEvaluateUnparsedTextResolvesAgainstTemplateBase proves a dynamic XPath
// evaluated by xsl:evaluate resolves a relative fn:unparsed-text href against
// the in-scope (template / xml:base) static base URI, exactly like the same
// call made statically. Before the fix xsl:evaluate hard-coded the MAIN
// stylesheet base, so a relative href inside an xml:base (or included-module)
// scope resolved to the wrong directory and missed the resolver.
func TestEvaluateUnparsedTextResolvesAgainstTemplateBase(t *testing.T) {
	// xml:base="sub/" against the compiler base "mem://pkg/main.xsl" yields the
	// template base "mem://pkg/sub/", so "data.txt" resolves under sub/.
	const wantURI = "mem://pkg/sub/data.txt"

	run := func(expr string) (*recordingURIResolver, string) {
		src := strings.ReplaceAll(evaluateXMLBaseStylesheet, "%EXPR%", expr)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().BaseURI("mem://pkg/main.xsl").Compile(t.Context(), doc)
		require.NoError(t, err)
		resolver := &recordingURIResolver{files: map[string][]byte{wantURI: []byte("payload")}}
		out, err := ss.Transform(parseTransformSource(t)).URIResolver(resolver).Serialize(t.Context())
		require.NoError(t, err)
		return resolver, out
	}

	// Baseline: the same call made statically resolves against the template base.
	staticResolver, staticOut := run(`<xsl:value-of select="unparsed-text($u)"/>`)
	require.Contains(t, staticOut, "<out>payload</out>")
	require.True(t, staticResolver.seen(wantURI),
		"static unparsed-text must resolve to %q; got %v", wantURI, staticResolver.requests)

	// Dynamic: xsl:evaluate must resolve against the SAME template base.
	dynResolver, dynOut := run(`<xsl:evaluate xpath="'unparsed-text($u)'"/>`)
	require.Contains(t, dynOut, "<out>payload</out>")
	require.True(t, dynResolver.seen(wantURI),
		"xsl:evaluate dynamic unparsed-text must resolve to %q like the static call; got %v",
		wantURI, dynResolver.requests)
}

// evaluateCodepointsStylesheet maps an XML 1.1 restricted character (U+0001),
// which is legal under an XSLT 3.0 processor (XML 1.1 chars allowed) but would
// raise FOCH0001 under a plain XML-1.0 evaluator. We measure string-length to
// avoid serializing the control character.
const evaluateCodepointsStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>%EXPR%</out>
  </xsl:template>
</xsl:stylesheet>`

// TestEvaluateCodepointsToStringAllowsXML11 proves a dynamic XPath evaluated by
// xsl:evaluate enables XML 1.1 restricted characters just like ordinary XSLT
// XPath evaluation. Before the fix the dynamic evaluator omitted
// AllowXML11Chars, so codepoints-to-string(1) raised FOCH0001 dynamically while
// succeeding statically.
func TestEvaluateCodepointsToStringAllowsXML11(t *testing.T) {
	run := func(expr string) (string, error) {
		src := strings.ReplaceAll(evaluateCodepointsStylesheet, "%EXPR%", expr)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		ss, err := xslt3.CompileStylesheet(t.Context(), doc)
		require.NoError(t, err)
		return ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	}

	// Baseline: the static call succeeds with XML 1.1 chars enabled.
	staticOut, err := run(`<xsl:value-of select="string-length(codepoints-to-string(1))"/>`)
	require.NoError(t, err)
	require.Contains(t, staticOut, "<out>1</out>")

	// Dynamic: xsl:evaluate must enable XML 1.1 chars too, matching the static path.
	dynOut, err := run(`<xsl:evaluate xpath="'string-length(codepoints-to-string(1))'"/>`)
	require.NoError(t, err)
	require.Contains(t, dynOut, "<out>1</out>")
}

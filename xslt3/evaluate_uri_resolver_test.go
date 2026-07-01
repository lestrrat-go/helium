package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// evaluateUnparsedTextStylesheet runs a dynamic XPath (via xsl:evaluate) that
// calls fn:unparsed-text() with a relative href. fn:unparsed-text is a plain
// XPath built-in (not overridden by xslt3), so it reads its URIResolver and
// resource cap straight from the evaluator the dynamic expression runs under.
// The %EXPR% placeholder lets a test run the SAME call statically for parity.
const evaluateUnparsedTextStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="u" select="'data.txt'"/>
  <xsl:template match="/">
    <out>%EXPR%</out>
  </xsl:template>
</xsl:stylesheet>`

func runUnparsedTextStylesheet(t *testing.T, expr string, resolver *recordingURIResolver) (string, error) {
	t.Helper()
	src := strings.ReplaceAll(evaluateUnparsedTextStylesheet, "%EXPR%", expr)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI("mem://pkg/main.xsl").Compile(t.Context(), doc)
	require.NoError(t, err)
	source := parseTransformSource(t)
	return ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
}

// TestEvaluateUnparsedTextRoutesThroughResolver proves a dynamic XPath
// evaluated by xsl:evaluate honors the runtime URIResolver and stylesheet base
// URI exactly like the same call made statically. Before the fix xsl:evaluate
// built a fresh evaluator that DROPPED the URIResolver (and other runtime
// config), so fn:unparsed-text inside the dynamic expression could not reach the
// resolver and failed, diverging from the static path.
func TestEvaluateUnparsedTextRoutesThroughResolver(t *testing.T) {
	const wantURI = "mem://pkg/data.txt"

	newResolver := func() *recordingURIResolver {
		return &recordingURIResolver{files: map[string][]byte{wantURI: []byte("payload")}}
	}

	// Baseline: the same call made statically routes through the resolver.
	staticResolver := newResolver()
	staticResult, err := runUnparsedTextStylesheet(t, `<xsl:value-of select="unparsed-text($u)"/>`, staticResolver)
	require.NoError(t, err)
	require.Contains(t, staticResult, "<out>payload</out>")
	require.True(t, staticResolver.seen(wantURI),
		"static unparsed-text must reach the resolver with %q; got %v", wantURI, staticResolver.requests)

	// Dynamic: the call routed through xsl:evaluate must behave identically.
	dynResolver := newResolver()
	dynResult, err := runUnparsedTextStylesheet(t, `<xsl:evaluate xpath="'unparsed-text($u)'"/>`, dynResolver)
	require.NoError(t, err)
	require.Contains(t, dynResult, "<out>payload</out>")
	require.True(t, dynResolver.seen(wantURI),
		"xsl:evaluate dynamic unparsed-text must reach the resolver with %q; got %v", wantURI, dynResolver.requests)
}

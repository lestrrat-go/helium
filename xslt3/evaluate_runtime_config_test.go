package xslt3_test

import (
	"io"
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

// fixedBasePackageResolver serves a single package source under a fixed module
// base URI (systemId). The base URI lets the test prove a package global
// resolves resources against the package's declaration/module base.
type fixedBasePackageResolver struct {
	source  string
	baseURI string
}

func (r fixedBasePackageResolver) ResolvePackage(_ string, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader(r.source)), r.baseURI, nil
}

// evaluateGlobalVarPackageSource declares a PUBLIC global variable whose body
// runs xsl:evaluate(unparsed-text(...)). The variable has no xml:base, so its
// declaration-site base is the package module base. Because it is a package
// component it is NOT eagerly evaluated; it is computed lazily on first
// reference, which happens from inside the using stylesheet's template (whose
// xml:base differs from the package base).
const evaluateGlobalVarPackageSource = `<?xml version="1.0"?>
<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="g" visibility="public">
    <xsl:evaluate xpath="'unparsed-text(&quot;data.txt&quot;)'"/>
  </xsl:variable>
</xsl:package>`

// evaluateGlobalVarUsingStylesheet references the package's public $g from a
// template carrying its own xml:base, so the lazily-evaluated package global is
// triggered while a DIFFERENT template base is current.
const evaluateGlobalVarUsingStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/pkg"/>
  <xsl:template match="/" xml:base="sub/">
    <out><xsl:value-of select="$g"/></out>
  </xsl:template>
</xsl:stylesheet>`

// TestEvaluateInLazyGlobalVarResolvesAgainstModuleBase proves a dynamic XPath
// evaluated by xsl:evaluate inside a LAZILY-evaluated global variable body
// resolves a relative fn:unparsed-text href against the global's
// declaration/module base, not the template that happened to trigger the lazy
// evaluation. Before the fix the lazy global cleared the static-base-URI
// override to "", so the dynamic evaluation fell through to the currently
// executing template's xml:base and missed the resolver.
func TestEvaluateInLazyGlobalVarResolvesAgainstModuleBase(t *testing.T) {
	// The package global's declaration site is the package module base
	// "mem://lib/lib.xsl"; "data.txt" must resolve to the package directory,
	// NOT the using template's xml:base ("mem://using/sub/").
	const moduleURI = "mem://lib/data.txt"
	const templateURI = "mem://using/sub/data.txt"

	doc, err := helium.NewParser().Parse(t.Context(), []byte(evaluateGlobalVarUsingStylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().
		BaseURI("mem://using/main.xsl").
		PackageResolver(fixedBasePackageResolver{source: evaluateGlobalVarPackageSource, baseURI: "mem://lib/lib.xsl"}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	resolver := &recordingURIResolver{files: map[string][]byte{moduleURI: []byte("payload")}}
	out, err := ss.Transform(parseTransformSource(t)).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<out>payload</out>")
	require.True(t, resolver.seen(moduleURI),
		"lazy global-body xsl:evaluate unparsed-text must resolve against the package module base %q; got %v",
		moduleURI, resolver.requests)
	require.False(t, resolver.seen(templateURI),
		"must NOT resolve against the referencing template's xml:base %q; got %v",
		templateURI, resolver.requests)
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

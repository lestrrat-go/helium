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

// evaluateEmptyBasePackageSource declares a PUBLIC global variable whose body
// runs xsl:evaluate(unparsed-text(...)). The variable has no xml:base and the
// package is served with an EMPTY module base, so its declaration-site base is
// the empty string. Like a normal package component it is evaluated lazily on
// first reference from the using stylesheet's template.
const evaluateEmptyBasePackageSource = `<?xml version="1.0"?>
<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="g" visibility="public">
    <xsl:evaluate xpath="'unparsed-text(&quot;data.txt&quot;)'"/>
  </xsl:variable>
</xsl:package>`

// TestEvaluateInLazyGlobalVarWithEmptyModuleBaseDoesNotFallThrough proves that a
// pinned but EMPTY declaration-site base is authoritative: the lazily-evaluated
// package global's xsl:evaluate/unparsed-text resolves the relative href against
// the empty declaration base, NOT the using stylesheet's non-empty base. With an
// absent base a relative href is unresolvable, so the correct outcome is a
// FOUT1170 "no base URI" failure and the resolver is NEVER asked for the using
// stylesheet's URI. Before the fix an empty pinned base fell through to
// ec.stylesheet.baseURI, so the global silently resolved against the USING
// stylesheet ("mem://using/data.txt") — the same class of bug this branch fixes.
func TestEvaluateInLazyGlobalVarWithEmptyModuleBaseDoesNotFallThrough(t *testing.T) {
	// If the bug were present the relative href would be joined against the
	// using stylesheet base and resolve to this URI; it must NOT.
	const usingURI = "mem://using/data.txt"

	doc, err := helium.NewParser().Parse(t.Context(), []byte(evaluateGlobalVarUsingStylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().
		BaseURI("mem://using/main.xsl").
		PackageResolver(fixedBasePackageResolver{source: evaluateEmptyBasePackageSource, baseURI: ""}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	// Register the using-base URI so that, were the bug present, the lookup
	// would succeed against it. The fix keeps the base empty, so resolution
	// fails before any resolver call with that URI.
	resolver := &recordingURIResolver{files: map[string][]byte{usingURI: []byte("payload")}}
	_, err = ss.Transform(parseTransformSource(t)).URIResolver(resolver).Serialize(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "without base URI",
		"empty declaration base must make the relative href unresolvable (FOUT1170), got %v", err)
	require.False(t, resolver.seen(usingURI),
		"must NOT fall through to the using stylesheet base %q; got %v",
		usingURI, resolver.requests)
}

// evaluateXMLBaseDocStylesheet mirrors evaluateXMLBaseStylesheet but exercises
// fn:doc instead of fn:unparsed-text: a relative href passed to doc() must
// resolve against the TEMPLATE base (its xml:base), not the main stylesheet
// base. fn:doc is an XSLT-aware wrapper that resolves through ec.baseDir(); this
// proves that path honors the effective static base inside xsl:evaluate too.
const evaluateXMLBaseDocStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="u" select="'data.xml'"/>
  <xsl:template match="/" xml:base="sub/">
    <out>%EXPR%</out>
  </xsl:template>
</xsl:stylesheet>`

// TestEvaluateDocResolvesAgainstTemplateBase proves a dynamic XPath evaluated by
// xsl:evaluate resolves a relative fn:doc href against the in-scope (template /
// xml:base) static base URI, exactly like the same call made statically. Before
// the fix fnDoc used ec.baseDir() which ignored the evaluator base, so a
// relative doc() inside an xml:base scope resolved to the wrong directory.
func TestEvaluateDocResolvesAgainstTemplateBase(t *testing.T) {
	// xml:base="sub/" against the compiler base "mem://pkg/main.xsl" yields the
	// template base "mem://pkg/sub/", so "data.xml" resolves under sub/.
	const wantURI = "mem://pkg/sub/data.xml"

	run := func(expr string) (*recordingURIResolver, string) {
		src := strings.ReplaceAll(evaluateXMLBaseDocStylesheet, "%EXPR%", expr)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().BaseURI("mem://pkg/main.xsl").Compile(t.Context(), doc)
		require.NoError(t, err)
		resolver := &recordingURIResolver{files: map[string][]byte{wantURI: []byte(`<data v="payload"/>`)}}
		out, err := ss.Transform(parseTransformSource(t)).URIResolver(resolver).Serialize(t.Context())
		require.NoError(t, err)
		return resolver, out
	}

	// Baseline: the same call made statically resolves against the template base.
	staticResolver, staticOut := run(`<xsl:value-of select="string(doc($u)/data/@v)"/>`)
	require.Contains(t, staticOut, "<out>payload</out>")
	require.True(t, staticResolver.seen(wantURI),
		"static doc() must resolve to %q; got %v", wantURI, staticResolver.requests)

	// Dynamic: xsl:evaluate must resolve against the SAME template base.
	dynResolver, dynOut := run(`<xsl:evaluate xpath="'string(doc($u)/data/@v)'"/>`)
	require.Contains(t, dynOut, "<out>payload</out>")
	require.True(t, dynResolver.seen(wantURI),
		"xsl:evaluate dynamic doc() must resolve to %q like the static call; got %v",
		wantURI, dynResolver.requests)
}

// evaluateDocBaseURIAttrStylesheet places a base-uri attribute on xsl:evaluate
// that differs from both the stylesheet and the template base. fn:doc inside the
// dynamic expression must resolve the relative href against the declared
// base-uri, not the template base.
const evaluateDocBaseURIAttrStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:evaluate xpath="'string(doc(&quot;data.xml&quot;)/data/@v)'" base-uri="mem://other/"/></out>
  </xsl:template>
</xsl:stylesheet>`

// TestEvaluateDocResolvesAgainstBaseURIAttribute proves the xsl:evaluate
// base-uri attribute governs XSLT-aware functions (fn:doc) and not just the
// native xpath3 functions: doc("data.xml") must resolve against "mem://other/",
// the declared base, rather than the using template's static base.
func TestEvaluateDocResolvesAgainstBaseURIAttribute(t *testing.T) {
	const overrideURI = "mem://other/data.xml"
	const templateURI = "mem://pkg/data.xml"

	doc, err := helium.NewParser().Parse(t.Context(), []byte(evaluateDocBaseURIAttrStylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI("mem://pkg/main.xsl").Compile(t.Context(), doc)
	require.NoError(t, err)

	resolver := &recordingURIResolver{files: map[string][]byte{overrideURI: []byte(`<data v="payload"/>`)}}
	out, err := ss.Transform(parseTransformSource(t)).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<out>payload</out>")
	require.True(t, resolver.seen(overrideURI),
		"xsl:evaluate doc() must resolve against the base-uri attribute %q; got %v",
		overrideURI, resolver.requests)
	require.False(t, resolver.seen(templateURI),
		"must NOT resolve against the using template base %q; got %v",
		templateURI, resolver.requests)
}

// evaluateGlobalVarDocPackageSource mirrors evaluateGlobalVarPackageSource but
// the lazily-evaluated public global runs xsl:evaluate(doc(...)). It proves the
// XSLT-aware fn:doc honors the global's pinned declaration/module base, not the
// template that triggered the lazy evaluation.
const evaluateGlobalVarDocPackageSource = `<?xml version="1.0"?>
<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="g" visibility="public">
    <xsl:evaluate xpath="'string(doc(&quot;data.xml&quot;)/data/@v)'"/>
  </xsl:variable>
</xsl:package>`

// TestEvaluateInLazyGlobalVarDocResolvesAgainstModuleBase proves a dynamic XPath
// evaluated by xsl:evaluate inside a LAZILY-evaluated global variable body
// resolves a relative fn:doc href against the global's declaration/module base,
// not the template that happened to trigger the lazy evaluation.
func TestEvaluateInLazyGlobalVarDocResolvesAgainstModuleBase(t *testing.T) {
	const moduleURI = "mem://lib/data.xml"
	const templateURI = "mem://using/sub/data.xml"

	doc, err := helium.NewParser().Parse(t.Context(), []byte(evaluateGlobalVarUsingStylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().
		BaseURI("mem://using/main.xsl").
		PackageResolver(fixedBasePackageResolver{source: evaluateGlobalVarDocPackageSource, baseURI: "mem://lib/lib.xsl"}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	resolver := &recordingURIResolver{files: map[string][]byte{moduleURI: []byte(`<data v="payload"/>`)}}
	out, err := ss.Transform(parseTransformSource(t)).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<out>payload</out>")
	require.True(t, resolver.seen(moduleURI),
		"lazy global-body xsl:evaluate doc() must resolve against the package module base %q; got %v",
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

// localFileBaseDocStylesheet calls doc() on a relative href so the resolved URI
// depends entirely on how the engine derives the document base directory from a
// local (no-scheme) Compiler.BaseURI.
const localFileBaseDocStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="string(doc('data.xml')/data/@v)"/></out>
  </xsl:template>
</xsl:stylesheet>`

// TestDocResolvesAgainstLocalFileBase pins the runtime document-base derivation
// for a LOCAL (extensionless) stylesheet FILE base to the same file-path
// semantics compile-time module resolution uses (path.Dir over the slashed
// base). Compiler.BaseURI("/styles/main") is a FILE path, so doc("data.xml")
// must resolve to its SIBLING "/styles/data.xml" — not to "/styles/main/data.xml"
// as a directory-treating heuristic would wrongly produce.
func TestDocResolvesAgainstLocalFileBase(t *testing.T) {
	const wantURI = "/styles/data.xml"

	doc, err := helium.NewParser().Parse(t.Context(), []byte(localFileBaseDocStylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI("/styles/main").Compile(t.Context(), doc)
	require.NoError(t, err)

	resolver := &recordingURIResolver{files: map[string][]byte{wantURI: []byte(`<data v="payload"/>`)}}
	out, err := ss.Transform(parseTransformSource(t)).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<out>payload</out>")
	require.True(t, resolver.seen(wantURI),
		"doc() against local file base /styles/main must resolve to %q; got %v",
		wantURI, resolver.requests)
}

package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestPatternPredeclaredFunctionNamespace verifies that match patterns may use
// the XPath 3.0 predeclared namespace prefixes (fn:, math:, map:, ...) without
// an explicit xmlns declaration in the stylesheet. The static context
// predeclares these bindings per XPath 3.0 / XSLT 3.0.
func TestPatternPredeclaredFunctionNamespace(t *testing.T) {
	t.Parallel()

	const tmpl = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"%s>
  <xsl:template match="%s"><out/></xsl:template>
</xsl:stylesheet>`

	tests := []struct {
		name    string
		extraNS string
		match   string
		wantErr bool
	}{
		{name: "fn-id-predeclared", match: "fn:id('x')"},
		{name: "id-unprefixed", match: "id('x')"},
		{
			name:    "fn-id-explicit-xmlns",
			extraNS: ` xmlns:fn="http://www.w3.org/2005/xpath-functions"`,
			match:   "fn:id('x')",
		},
		{name: "math-predeclared-predicate", match: "*[math:pi() > 3]"},
		{name: "math-sqrt-predeclared", match: "*[math:sqrt(4) = 2]"},
		{name: "map-predeclared-predicate", match: "*[map:size(map{}) = 0]"},
		{
			name:    "unknown-prefix-fails",
			match:   "bogus:id('x')",
			wantErr: true,
		},
		{
			name:    "math-unknown-function-fails",
			match:   "*[math:no-such-function()]",
			wantErr: true,
		},
		{
			name:    "fn-unknown-function-fails",
			match:   "*[fn:no-such-function()]",
			wantErr: true,
		},
		{
			name:    "map-unknown-function-fails",
			match:   "*[map:no-such-function()]",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := strings.Replace(tmpl, "%s", tc.extraNS, 1)
			src = strings.Replace(src, "%s", tc.match, 1)

			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)

			_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
			if tc.wantErr {
				// An undeclared prefix must still be rejected at compile time
				// (XPST0081 at prefix resolution / XPST0017 at function check).
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestPatternPredeclaredFunctionMatchesAtRuntime verifies that a template whose
// match pattern relies on a predeclared XPath namespace prefix (math:, fn:)
// without an explicit xmlns declaration not only compiles but actually MATCHES
// at runtime. Compile-time and runtime prefix resolution must be symmetric:
// if the pattern compiles, it must also be evaluable.
func TestPatternPredeclaredFunctionMatchesAtRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		match string
	}{
		{name: "math-predicate", match: "a[math:pi() > 3]"},
		{name: "math-sqrt", match: "a[math:sqrt(4) = 2]"},
		{name: "fn-predicate", match: "a[fn:true()]"},
		{name: "map-predicate", match: "a[map:size(map{}) = 0]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="root/a"/></out></xsl:template>
  <xsl:template match="` + tc.match + `">[matched]</xsl:template>
</xsl:stylesheet>`

			doc, err := helium.NewParser().Parse(t.Context(), []byte(xsltSrc))
			require.NoError(t, err)
			ss, err := xslt3.CompileStylesheet(t.Context(), doc)
			require.NoError(t, err)

			src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a>x</a></root>`))
			require.NoError(t, err)

			out, err := ss.Transform(src).Serialize(t.Context())
			require.NoError(t, err)
			// The specialized template must win over the built-in template, so
			// the predeclared-prefix predicate must evaluate true at runtime.
			require.Contains(t, out, "[matched]", "pattern %q must match at runtime", tc.match)
		})
	}
}

// TestPatternXSLTFunctionAllowed verifies that XSLT-defined functions in the fn
// namespace (key, current, document, ...) are accepted in match patterns —
// compile-time validation must consult the XSLT function registry, not only the
// XPath built-in registry. These previously raised a spurious XPST0017.
func TestPatternXSLTFunctionAllowed(t *testing.T) {
	t.Parallel()

	// fn:key in a pattern: the template matches nodes returned by the key.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="k" match="a" use="@id"/>
  <xsl:template match="/"><out><xsl:apply-templates select="root/a"/></out></xsl:template>
  <xsl:template match="fn:key('k', '1')">[keyed]</xsl:template>
  <xsl:template match="a">[plain]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a id="1">x</a><a id="2">y</a></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "[keyed]")
	require.Contains(t, out, "[plain]")
}

// TestPatternUnprefixedFunctionValidation verifies that unprefixed function
// calls in patterns get the same existence/arity validation as their fn:-prefixed
// forms. An unprefixed call names the XPath functions namespace, so a nonexistent
// function must be rejected (XPST0017) and a wrong-arity call must not escape to
// runtime.
func TestPatternUnprefixedFunctionValidation(t *testing.T) {
	t.Parallel()

	const tmpl = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="k" match="a" use="@id"/>
  <xsl:template match="%s"><out/></xsl:template>
</xsl:stylesheet>`

	tests := []struct {
		name    string
		match   string
		wantErr bool
	}{
		{name: "unprefixed-unknown-function", match: "*[no-such-function()]", wantErr: true},
		{name: "unprefixed-wrong-arity-key", match: "key('k')", wantErr: true},
		{name: "unprefixed-correct-arity-key", match: "key('k', '1')"},
		{name: "unprefixed-known-function", match: "*[true()]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := strings.Replace(tmpl, "%s", tc.match, 1)
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)

			_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestPatternForbiddenFunctionRespectsExplicitBinding verifies that the
// forbidden-in-pattern check (current-group, current-merge-key, ...) is applied
// only when the call actually resolves to the XPath functions namespace. An
// explicit xmlns:fn override pointing at a custom namespace means fn:current-group
// is a user function, not the XSLT builtin, so it must NOT be rejected. A real
// fn:current-group() (no override) must still be forbidden.
func TestPatternForbiddenFunctionRespectsExplicitBinding(t *testing.T) {
	t.Parallel()

	// fn rebound to a custom namespace: fn:current-group() is a user function
	// declared as xsl:function. It must compile (not be rejected as the builtin).
	overrideSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:fn="urn:custom">
  <xsl:function name="fn:current-group" as="xs:boolean">
    <xsl:sequence select="true()"/>
  </xsl:function>
  <xsl:template match="a[fn:current-group()]"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(overrideSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err, "fn:current-group() in custom namespace must not be rejected as the XSLT builtin")

	// No override: real fn:current-group() in a pattern is forbidden (XTSE1060).
	realSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a[fn:current-group()]"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err = helium.NewParser().Parse(t.Context(), []byte(realSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.Error(t, err, "real fn:current-group() in a pattern must be forbidden")

	// Unprefixed current-group() resolves to the XPath functions namespace and
	// must also be forbidden.
	plainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a[current-group()]"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err = helium.NewParser().Parse(t.Context(), []byte(plainSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.Error(t, err, "unprefixed current-group() in a pattern must be forbidden")
}

// TestPatternLexicalNamespaceSnapshot verifies that each match pattern resolves
// prefixes against its OWN lexical namespace context (the xmlns declarations in
// scope at the pattern's position), not against a mutable stylesheet-global map.
// An unrelated earlier top-level declaration carrying xmlns:math="urn:custom"
// must NOT leak into a later template whose match pattern has no local
// xmlns:math — there, math: must resolve to the predeclared XPath math namespace
// (http://www.w3.org/2005/xpath-functions/math) so math:pi() is valid and the
// pattern matches. A pattern that DOES carry a local xmlns:math override uses
// that override (and consequently rejects a builtin-only call).
func TestPatternLexicalNamespaceSnapshot(t *testing.T) {
	t.Parallel()

	// An earlier top-level xsl:variable carries xmlns:math="urn:custom". The
	// later template has NO xmlns:math, so its pattern must resolve math: to the
	// predeclared math namespace, NOT the leaked urn:custom.
	leakSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="v" xmlns:math="urn:custom" select="1"/>
  <xsl:template match="/"><out><xsl:apply-templates select="root/a"/></out></xsl:template>
  <xsl:template match="a[math:pi() > 3]">[matched]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(leakSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err, "earlier xmlns:math='urn:custom' must not leak into a later pattern's math: prefix")

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a>x</a></root>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "[matched]",
		"math: must resolve to the predeclared math namespace, not leaked urn:custom")

	// A pattern WITH a local xmlns:math override binds math: to urn:x. There
	// math:pi() is a user-namespace function that is not declared, so compilation
	// must reject it (explicit lexical binding wins over the predeclared fallback).
	overrideSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a[math:pi() > 3]" xmlns:math="urn:x">[matched]</xsl:template>
</xsl:stylesheet>`

	doc, err = helium.NewParser().Parse(t.Context(), []byte(overrideSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.Error(t, err,
		"a local xmlns:math override must bind math: to urn:x, making math:pi() an unknown function")
}

// TestPatternFnCurrentCompiles verifies fn:current() is accepted (not rejected
// as XPST0017) inside a pattern predicate.
func TestPatternFnCurrentCompiles(t *testing.T) {
	t.Parallel()

	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a[fn:current()]">[matched]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xsltSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
}

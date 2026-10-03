package xslt3_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// modeAllPackageResolver serves a single fixed package source for any name.
type modeAllPackageResolver struct {
	source string
}

func (r modeAllPackageResolver) ResolvePackage(_ string, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader(r.source)), "", nil
}

// A used package exposes a public template match="b" mode="#all". The using
// stylesheet applies templates to <b> in a named mode "m"; the package template
// must be eligible (via the #all fallback) even though "m" is not the default
// mode.
func TestUsePackageModeAllTemplateAppliedInNamedMode(t *testing.T) {
	t.Parallel()

	pkg := `<?xml version="1.0"?>
<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"
             declared-modes="no"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:expose component="mode" names="*" visibility="public"/>
  <xsl:template match="b" mode="#all">
    <hit-all/>
  </xsl:template>
</xsl:package>`

	using := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/pkg"/>
  <xsl:template match="/">
    <out>
      <named><xsl:apply-templates select="//b" mode="m"/></named>
      <default><xsl:apply-templates select="//b"/></default>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(using))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		PackageResolver(modeAllPackageResolver{source: pkg}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><b/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)

	// The #all package template must fire in both the named mode "m" and the
	// default mode.
	require.Equal(t, 2, strings.Count(result, "<hit-all"),
		"package template match=\"b\" mode=\"#all\" should fire in named mode and default mode; got: %s", result)
}

// overrideUnionPackage is a used package with two public modes. Its own
// template rules only report "pkg", so any other output comes from the using
// stylesheet's xsl:override rules.
const overrideUnionPackage = `<?xml version="1.0"?>
<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode name="m" visibility="public"/>
  <xsl:mode name="n" visibility="public"/>
  <xsl:template match="*" mode="m n">[pkg]</xsl:template>
</xsl:package>`

// overrideUnionSource is the source document runOverrideUnion transforms.
const overrideUnionSource = `<doc><a n="2"/><b><c n="2"/></b></doc>`

// runOverrideUnion compiles a using stylesheet whose xsl:override holds
// overrides, then applies templates in mode m (and in mode n when applyN is
// set) to the doc/a and doc/b/c elements of overrideUnionSource.
func runOverrideUnion(t *testing.T, overrides string, applyN bool) string {
	t.Helper()

	apply := `<m><xsl:apply-templates select="doc/a | doc/b/c" mode="m"/></m>`
	if applyN {
		apply += `<n><xsl:apply-templates select="doc/a | doc/b/c" mode="n"/></n>`
	}
	using := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/pkg">
    <xsl:override>` + overrides + `</xsl:override>
  </xsl:use-package>
  <xsl:template match="/"><out>` + apply + `</out></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(using))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		PackageResolver(modeAllPackageResolver{source: overrideUnionPackage}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(t.Context(), []byte(overrideUnionSource))
	require.NoError(t, err)

	result, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)
	return result
}

// An xsl:override template rule with a union match pattern and no priority is
// one template rule per alternative, each with that alternative's default
// priority (XSLT 3.0 §6.4, §6.5): "a" gets 0 and "b/c" gets 0.5. A competing
// override rule at priority 0.25 therefore loses to the union on <c> and wins
// on <a>. An explicit priority applies to every alternative.
func TestOverrideUnionTemplatePriority(t *testing.T) {
	t.Parallel()

	const competitors = `
      <xsl:template match="a" mode="m" priority="0.25">[a-0.25]</xsl:template>
      <xsl:template match="c" mode="m" priority="0.25">[c-0.25]</xsl:template>`

	t.Run("default priority per alternative", func(t *testing.T) {
		t.Parallel()
		overrides := `<xsl:template match="a | b/c" mode="m">[union]</xsl:template>` + competitors
		result := runOverrideUnion(t, overrides, false)
		require.Contains(t, result, "<m>[a-0.25][union]</m>")
	})

	t.Run("explicit priority applies to all alternatives", func(t *testing.T) {
		t.Parallel()
		overrides := `<xsl:template match="a | b/c" mode="m" priority="0.3">[union]</xsl:template>` + competitors
		result := runOverrideUnion(t, overrides, false)
		require.Contains(t, result, "<m>[union][union]</m>")
	})

	// Under version="1.0" the predicate @n = true() converts @n to a boolean
	// (true for any present attribute), so both split rules match. Under
	// version="3.0" casting "2" to xs:boolean fails, so neither matches.
	t.Run("backwards-compatible processing", func(t *testing.T) {
		t.Parallel()
		const union = `<xsl:template match="a[@n = true()] | b/c[@n = true()]" mode="m" version="%s">[union]</xsl:template>`
		require.Contains(t, runOverrideUnion(t, fmt.Sprintf(union, "1.0"), false), "<m>[union][union]</m>")
		require.Contains(t, runOverrideUnion(t, fmt.Sprintf(union, "3.0"), false), "<m>[pkg][pkg]</m>")
	})

	t.Run("mode list", func(t *testing.T) {
		t.Parallel()
		overrides := `<xsl:template match="a | b/c" mode="m n">[union]</xsl:template>
      <xsl:template match="c" mode="m n" priority="0.25">[c-0.25]</xsl:template>`
		result := runOverrideUnion(t, overrides, true)
		require.Contains(t, result, "<m>[union][union]</m><n>[union][union]</n>")
	})
}

// A used package's own union template rule without a priority reaches the
// using stylesheet as one rule per alternative with that alternative's default
// priority, exactly as it does inside the package.
func TestUsePackageUnionTemplatePriority(t *testing.T) {
	t.Parallel()

	pkg := `<?xml version="1.0"?>
<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode name="m" visibility="public"/>
  <xsl:template match="a | b/c" mode="m">[union]</xsl:template>
  <xsl:template match="a" mode="m" priority="0.25">[a-0.25]</xsl:template>
  <xsl:template match="c" mode="m" priority="0.25">[c-0.25]</xsl:template>
</xsl:package>`

	using := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/pkg"/>
  <xsl:template match="/"><out><xsl:apply-templates select="doc/a | doc/b/c" mode="m"/></out></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(using))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		PackageResolver(modeAllPackageResolver{source: pkg}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<doc><a/><b><c/></b></doc>`))
	require.NoError(t, err)

	result, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>[a-0.25][union]</out>")
}

// namedPackageResolver serves package sources by package name.
type namedPackageResolver map[string]string

func (r namedPackageResolver) ResolvePackage(name string, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader(r[name])), "", nil
}

// An xsl:override template with both a name and a union match pattern, in a
// package that is itself used by the top-level stylesheet, contributes each
// alternative once. Under on-multiple-match="fail" the alternatives of that
// one rule never conflict with each other (spec bug 30402).
func TestUsePackageNestedOverrideUnionRule(t *testing.T) {
	t.Parallel()

	inner := `<?xml version="1.0"?>
<xsl:package name="http://example.com/inner" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode name="m" visibility="public" on-multiple-match="fail"/>
  <xsl:template name="t" match="x" mode="m" visibility="public">[inner]</xsl:template>
</xsl:package>`

	outer := `<?xml version="1.0"?>
<xsl:package name="http://example.com/outer" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/inner">
    <xsl:override>
      <xsl:template name="t" match="a | b/c" mode="m">[union]</xsl:template>
    </xsl:override>
  </xsl:use-package>
</xsl:package>`

	using := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/outer"/>
  <xsl:template match="/"><out><xsl:apply-templates select="doc/a | doc/b/c" mode="m"/></out></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(using))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		PackageResolver(namedPackageResolver{
			"http://example.com/inner": inner,
			"http://example.com/outer": outer,
		}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<doc><a/><b><c/></b></doc>`))
	require.NoError(t, err)

	result, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>[union][union]</out>")
}

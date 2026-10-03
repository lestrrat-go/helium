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

// overrideDuplicatesPackage exposes one public component of each kind an
// xsl:override can replace.
const overrideDuplicatesPackage = `<?xml version="1.0"?>
<xsl:package name="%s" package-version="1.0" version="3.0"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="t" visibility="public">[t]</xsl:template>
  <xsl:variable name="v" select="1" visibility="public"/>
  <xsl:param name="p" select="1"/>
  <xsl:attribute-set name="s" visibility="public"><xsl:attribute name="x">1</xsl:attribute></xsl:attribute-set>
</xsl:package>`

// errXTSE3055 is the error code for homonymous overriding declarations.
const errXTSE3055 = "XTSE3055"

// XSLT 3.0 §3.5.3.2 (XTSE3055): a declaration inside xsl:override must not be
// homonymous with any other overriding declaration of the using package,
// whether both sit in the same xsl:use-package or in two of them.
func TestOverrideDuplicateDeclarations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		first  string
		second string
		// samePackage puts both overrides under one xsl:use-package; otherwise
		// each overrides a component of its own used package.
		samePackage bool
		code        string
	}{
		{
			name:        "distinct names",
			first:       `<xsl:template name="t">[1]</xsl:template>`,
			second:      `<xsl:variable name="v" select="2"/>`,
			samePackage: true,
		},
		{
			name:        "template twice",
			first:       `<xsl:template name="t">[1]</xsl:template>`,
			second:      `<xsl:template name="t">[2]</xsl:template>`,
			samePackage: true,
			code:        errXTSE3055,
		},
		{
			name:        "variable twice",
			first:       `<xsl:variable name="v" select="2"/>`,
			second:      `<xsl:variable name="v" select="3"/>`,
			samePackage: true,
			code:        errXTSE3055,
		},
		{
			name:        "param twice",
			first:       `<xsl:param name="p" select="2"/>`,
			second:      `<xsl:param name="p" select="3"/>`,
			samePackage: true,
			code:        errXTSE3055,
		},
		{
			name:        "attribute set twice",
			first:       `<xsl:attribute-set name="s"><xsl:attribute name="x">2</xsl:attribute></xsl:attribute-set>`,
			second:      `<xsl:attribute-set name="s"><xsl:attribute name="x">3</xsl:attribute></xsl:attribute-set>`,
			samePackage: true,
			code:        errXTSE3055,
		},
		{
			name:   "template in two used packages",
			first:  `<xsl:template name="t">[1]</xsl:template>`,
			second: `<xsl:template name="t">[2]</xsl:template>`,
			code:   errXTSE3055,
		},
		{
			name:        "variable and param with one name",
			first:       `<xsl:variable name="v" select="2"/>`,
			second:      `<xsl:param name="v" select="3"/>`,
			samePackage: true,
			code:        errXTSE3055,
		},
		{
			name:        "second template excluded by use-when",
			first:       `<xsl:template name="t">[1]</xsl:template>`,
			second:      `<xsl:template name="t" use-when="false()">[2]</xsl:template>`,
			samePackage: true,
		},
		{
			name:   "param in two used packages",
			first:  `<xsl:param name="p" select="2"/>`,
			second: `<xsl:param name="p" select="3"/>`,
			code:   errXTSE3055,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var uses string
			if tc.samePackage {
				uses = `<xsl:use-package name="urn:p1"><xsl:override>` + tc.first + tc.second + `</xsl:override></xsl:use-package>`
			} else {
				uses = `<xsl:use-package name="urn:p1"><xsl:accept component="*" names="*" visibility="hidden"/>` +
					`<xsl:override>` + tc.first + `</xsl:override></xsl:use-package>` +
					`<xsl:use-package name="urn:p2"><xsl:accept component="*" names="*" visibility="hidden"/>` +
					`<xsl:override>` + tc.second + `</xsl:override></xsl:use-package>`
			}
			using := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + uses + `
  <xsl:template name="xsl:initial-template"><out/></xsl:template>
</xsl:stylesheet>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(using))
			require.NoError(t, err)

			_, err = xslt3.NewCompiler().
				PackageResolver(namedPackageResolver{
					"urn:p1": fmt.Sprintf(overrideDuplicatesPackage, "urn:p1"),
					"urn:p2": fmt.Sprintf(overrideDuplicatesPackage, "urn:p2"),
				}).
				Compile(t.Context(), doc)
			if tc.code == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.code)
		})
	}
}

// use-when="false()" on xsl:override, or on one of its children, removes
// that element before compilation: the excluded declaration neither replaces
// the used package's component nor counts as an overriding declaration.
func TestOverrideUseWhen(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		override string
		want     string
	}{
		{
			name:     "included",
			override: `<xsl:override><xsl:template name="t">[1]</xsl:template></xsl:override>`,
			want:     "<out>[1]</out>",
		},
		{
			name:     "excluded child",
			override: `<xsl:override><xsl:template name="t" use-when="false()">[1]</xsl:template></xsl:override>`,
			want:     "<out>[t]</out>",
		},
		{
			name: "one of two homonymous children excluded",
			override: `<xsl:override><xsl:template name="t">[1]</xsl:template>` +
				`<xsl:template name="t" use-when="false()">[2]</xsl:template></xsl:override>`,
			want: "<out>[1]</out>",
		},
		{
			name:     "excluded override",
			override: `<xsl:override use-when="false()"><xsl:template name="t">[1]</xsl:template></xsl:override>`,
			want:     "<out>[t]</out>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			using := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="urn:p1">` + tc.override + `</xsl:use-package>
  <xsl:template match="/"><out><xsl:call-template name="t"/></out></xsl:template>
</xsl:stylesheet>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(using))
			require.NoError(t, err)

			ss, err := xslt3.NewCompiler().
				PackageResolver(namedPackageResolver{"urn:p1": fmt.Sprintf(overrideDuplicatesPackage, "urn:p1")}).
				Compile(t.Context(), doc)
			require.NoError(t, err)

			src, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
			require.NoError(t, err)

			result, err := ss.Transform(src).Serialize(t.Context())
			require.NoError(t, err)
			require.Contains(t, result, tc.want)
		})
	}
}

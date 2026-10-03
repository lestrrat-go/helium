package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// transformStr compiles the stylesheet and transforms <doc/>, returning the
// serialized result.
func transformStr(t *testing.T, xsltSrc string) (string, error) {
	t.Helper()
	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	if err != nil {
		return "", err
	}
	src, err := helium.NewParser().Parse(ctx, []byte(`<doc/>`))
	require.NoError(t, err)
	return ss.Transform(src).Serialize(ctx)
}

func TestBackCompat(t *testing.T) {
	// TestBackCompatXPathArithmetic verifies that a version="1.0" stylesheet both
	// compiles/runs (no XTDE0160) and evaluates XPath in 1.0 compatibility mode:
	// '3' + 4 becomes 7, and never a type error.
	t.Run("XPath arithmetic", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="'3' + 4"/></out></xsl:template>
</xsl:stylesheet>`
		out, err := transformStr(t, ss)
		require.NoError(t, err)
		require.Contains(t, out, "<out>7</out>")
	})

	// TestBackCompatVersionGated verifies the same expression is a type error under
	// version="3.0" — compatibility mode is opt-in via the version attribute.
	t.Run("version gated", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="'3' + 4"/></out></xsl:template>
</xsl:stylesheet>`
		_, err := transformStr(t, ss)
		require.Error(t, err)
	})

	// TestBackCompatPerSubtreeOverride verifies that an inner xsl:version on a
	// literal result element enables compatibility mode for that subtree only.
	t.Run("per subtree override", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out xsl:version="1.0"><xsl:value-of select="'3' + 4"/></out>
  </xsl:template>
</xsl:stylesheet>`
		out, err := transformStr(t, ss)
		require.NoError(t, err)
		require.Contains(t, out, "7")
	})

	// TestBackCompatSortKeyInnerCompat verifies that a version="1.0" xsl:sort key
	// evaluates its inner expression in XPath 1.0 compatibility mode even on the
	// optimized (EvaluateReuse) sort path: string(.) + 0 coerces the string to a
	// number, raising no XPTY0004.
	t.Run("sort key inner compat", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:for-each select="doc/i"><xsl:sort select="string(.) + 0"/><xsl:value-of select="."/></xsl:for-each></out></xsl:template>
</xsl:stylesheet>`
		ctx := t.Context()
		doc, err := helium.NewParser().Parse(ctx, []byte(ss))
		require.NoError(t, err)
		ssc, err := xslt3.CompileStylesheet(ctx, doc)
		require.NoError(t, err)
		src, err := helium.NewParser().Parse(ctx, []byte(`<doc><i>3</i><i>1</i><i>2</i></doc>`))
		require.NoError(t, err)
		out, err := ssc.Transform(src).Serialize(ctx)
		require.NoError(t, err)
		require.Contains(t, out, "<out>123</out>")
	})

	// TestBackCompatPatternPredicate verifies that a match-pattern predicate in a
	// version="1.0" stylesheet evaluates in XPath 1.0 compatibility mode: the
	// relational predicate . > '2' compares numerically (matching 3 and 10), not as
	// strings (which would match only 3).
	t.Run("pattern predicate", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="doc/i"/></out></xsl:template>
  <xsl:template match="i[. > '2']">M</xsl:template>
  <xsl:template match="i">-</xsl:template>
</xsl:stylesheet>`
		ctx := t.Context()
		doc, err := helium.NewParser().Parse(ctx, []byte(ss))
		require.NoError(t, err)
		ssc, err := xslt3.CompileStylesheet(ctx, doc)
		require.NoError(t, err)
		src, err := helium.NewParser().Parse(ctx, []byte(`<doc><i>1</i><i>2</i><i>3</i><i>10</i></doc>`))
		require.NoError(t, err)
		out, err := ssc.Transform(src).Serialize(ctx)
		require.NoError(t, err)
		require.Contains(t, out, "<out>--MM</out>")
	})

	// TestBackCompatTemplateVersionPattern verifies that a version="1.0" attribute on
	// a template (in a 3.0 module) makes its match-pattern predicate evaluate in
	// XPath 1.0 compatibility mode — the match pattern is compiled under the
	// template's own effective version.
	t.Run("template version pattern", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="doc/i"/></out></xsl:template>
  <xsl:template match="i[. > '2']" version="1.0">M</xsl:template>
  <xsl:template match="i">-</xsl:template>
</xsl:stylesheet>`
		ctx := t.Context()
		doc, err := helium.NewParser().Parse(ctx, []byte(ss))
		require.NoError(t, err)
		ssc, err := xslt3.CompileStylesheet(ctx, doc)
		require.NoError(t, err)
		src, err := helium.NewParser().Parse(ctx, []byte(`<doc><i>1</i><i>2</i><i>3</i><i>10</i></doc>`))
		require.NoError(t, err)
		out, err := ssc.Transform(src).Serialize(ctx)
		require.NoError(t, err)
		require.Contains(t, out, "<out>--MM</out>")
	})

	// A union match pattern is split into one template rule per alternative.
	// Every split rule keeps the pattern's backwards-compatible processing, so
	// under version="1.0" the predicate @n = true() converts @n to a boolean
	// (true for any present attribute) exactly as the non-union form does.
	// Under version="3.0" the comparison casts "2" to xs:boolean, which fails,
	// so neither form matches.
	t.Run("union pattern predicate", func(t *testing.T) {
		const unionRules = `<xsl:template match="a[@n = true()] | b[@n = true()]">[<xsl:value-of select="name()"/>]</xsl:template>`
		const separateRules = `<xsl:template match="a[@n = true()]">[<xsl:value-of select="name()"/>]</xsl:template>
  <xsl:template match="b[@n = true()]">[<xsl:value-of select="name()"/>]</xsl:template>`
		cases := []struct {
			name    string
			version string
			rules   string
			want    string
		}{
			{name: "union 1.0", version: "1.0", rules: unionRules, want: "<out>[a][b]</out>"},
			{name: "separate 1.0", version: "1.0", rules: separateRules, want: "<out>[a][b]</out>"},
			{name: "union 3.0", version: "3.0", rules: unionRules, want: "<out>--</out>"},
			{name: "separate 3.0", version: "3.0", rules: separateRules, want: "<out>--</out>"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				ss := `<?xml version="1.0"?>
<xsl:stylesheet version="` + tc.version + `" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="doc/*"/></out></xsl:template>
  ` + tc.rules + `
  <xsl:template match="*">-</xsl:template>
</xsl:stylesheet>`
				ctx := t.Context()
				doc, err := helium.NewParser().Parse(ctx, []byte(ss))
				require.NoError(t, err)
				ssc, err := xslt3.CompileStylesheet(ctx, doc)
				require.NoError(t, err)
				src, err := helium.NewParser().Parse(ctx, []byte(`<doc><a n="2"/><b n="2"/></doc>`))
				require.NoError(t, err)
				out, err := ssc.Transform(src).Serialize(ctx)
				require.NoError(t, err)
				require.Contains(t, out, tc.want)
			})
		}
	})

	// TestBackCompatLREUnqualifiedVersionNotCompat verifies that an unqualified
	// version attribute on a literal result element is an ordinary result attribute
	// (copied to output), NOT the XSLT version — so it does NOT trigger
	// backwards-compatible processing. Here '3' + 4 stays a type error.
	t.Run("LRE unqualified version not compat", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out version="1.0"><xsl:value-of select="'3' + 4"/></out></xsl:template>
</xsl:stylesheet>`
		_, err := transformStr(t, ss)
		require.Error(t, err) // '3' + 4 is XPTY0004 in 3.0; would be 7 if compat leaked

		// And the unqualified version is preserved as a literal result attribute.
		ss2 := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out version="1.0">hi</out></xsl:template>
</xsl:stylesheet>`
		out, err := transformStr(t, ss2)
		require.NoError(t, err)
		require.Contains(t, out, `version="1.0"`)
	})

	// TestBackCompatSupportsProperty verifies system-property reports support.
	t.Run("supports property", func(t *testing.T) {
		ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="system-property('xsl:supports-backwards-compatibility')"/></out></xsl:template>
</xsl:stylesheet>`
		out, err := transformStr(t, ss)
		require.NoError(t, err)
		require.Contains(t, out, "<out>yes</out>")
	})
}

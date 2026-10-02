package xslt3_test

import (
	"io"
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// identityTemplateStylesheet is the classic XSLT 1.0 identity transform: one
// template rule that matches every attribute and node, copies it, and
// recurses into its attributes and children.
const identityTemplateStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="@*|node()">
    <xsl:copy><xsl:apply-templates select="@*|node()"/></xsl:copy>
  </xsl:template>
</xsl:stylesheet>`

// identityModeStylesheet is the XSLT 3.0 identity transform: no template
// rules, and the unnamed mode's built-in rules shallow-copy every node.
const identityModeStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode on-no-match="shallow-copy"/>
</xsl:stylesheet>`

// BenchmarkIdentityTransform measures a whole identity transform over a real
// 608 KB document (testdata/libxml2-compat/relaxng/test/comps_0.xml), in the
// two forms a stylesheet author writes it:
//
//   - template: the explicit match="@*|node()" rule, which evaluates the
//     apply-templates select expression once per copied node.
//   - mode: xsl:mode on-no-match="shallow-copy", which copies through the
//     built-in rules with no user expression to evaluate.
//
// Each form has two sub-benchmarks. "writer" times xslt3.TransformToWriter
// into io.Discard, so it covers the transform plus serialization. "tree"
// times xslt3.Transform, which builds the result tree only; the difference
// between the two is the serialization cost.
//
// The source is parsed and the stylesheets are compiled once, outside the
// timed loop. A transform does not mutate its source, so every iteration
// reuses the same parsed document.
//
// Run it with:
//
//	go test ./xslt3 -run '^$' -bench BenchmarkIdentityTransform -benchmem -count=6
func BenchmarkIdentityTransform(b *testing.B) {
	srcBytes, err := os.ReadFile(heliumtest.TestDir("testdata", "libxml2-compat", "relaxng", "test", "comps_0.xml"))
	require.NoError(b, err)

	source, err := helium.NewParser().Parse(b.Context(), srcBytes)
	require.NoError(b, err)

	forms := []struct {
		name string
		ss   *xslt3.Stylesheet
	}{
		{name: "template", ss: compileBenchStylesheet(b, identityTemplateStylesheet)},
		{name: "mode", ss: compileBenchStylesheet(b, identityModeStylesheet)},
	}

	for _, form := range forms {
		b.Run(form.name+"/writer", func(b *testing.B) {
			b.SetBytes(int64(len(srcBytes)))
			b.ReportAllocs()
			for b.Loop() {
				if err := xslt3.TransformToWriter(b.Context(), source, form.ss, io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(form.name+"/tree", func(b *testing.B) {
			b.SetBytes(int64(len(srcBytes)))
			b.ReportAllocs()
			for b.Loop() {
				if _, err := xslt3.Transform(b.Context(), source, form.ss); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// conditionalPathStylesheet visits every package and tests node paths on
// their own with xsl:if and xsl:when.
const conditionalPathStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="//package"/></out></xsl:template>
  <xsl:template match="package">
    <xsl:if test="dependencylist/dependency"><deps/></xsl:if>
    <xsl:choose>
      <xsl:when test=".//nosuch"><nosuch/></xsl:when>
      <xsl:when test=".//dependency"><some/></xsl:when>
      <xsl:otherwise><none/></xsl:otherwise>
    </xsl:choose>
  </xsl:template>
</xsl:stylesheet>`

// conditionalExistsStylesheet is conditionalPathStylesheet with every test
// written as exists(), empty() or count() > 0 of the path.
const conditionalExistsStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="//package"/></out></xsl:template>
  <xsl:template match="package">
    <xsl:if test="exists(dependencylist/dependency)"><deps/></xsl:if>
    <xsl:choose>
      <xsl:when test="not(empty(.//nosuch))"><nosuch/></xsl:when>
      <xsl:when test="count(.//dependency) > 0"><some/></xsl:when>
      <xsl:otherwise><none/></xsl:otherwise>
    </xsl:choose>
  </xsl:template>
</xsl:stylesheet>`

// conditionalDocumentStylesheet visits every group and tests paths over the
// whole document, on their own and through exists().
const conditionalDocumentStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="//group"/></out></xsl:template>
  <xsl:template match="group">
    <xsl:if test="//packagereq"><pkgs/></xsl:if>
    <xsl:if test="exists(/comps/group/packagelist/packagereq)"><listed/></xsl:if>
  </xsl:template>
</xsl:stylesheet>`

// BenchmarkConditionalNodeTests measures transforms whose xsl:if and
// xsl:when tests only need to know whether a node path selects a node, over
// the same 608 KB document as BenchmarkIdentityTransform:
//
//   - path: per package, paths on their own as the test;
//   - exists: per package, the same paths through exists(), empty() and
//     count() > 0;
//   - document: per group, paths over the whole document.
//
// Each times xslt3.Transform, which builds the result tree only.
func BenchmarkConditionalNodeTests(b *testing.B) {
	srcBytes, err := os.ReadFile(heliumtest.TestDir("testdata", "libxml2-compat", "relaxng", "test", "comps_0.xml"))
	require.NoError(b, err)

	source, err := helium.NewParser().Parse(b.Context(), srcBytes)
	require.NoError(b, err)

	forms := []struct {
		name string
		ss   *xslt3.Stylesheet
	}{
		{name: "path", ss: compileBenchStylesheet(b, conditionalPathStylesheet)},
		{name: "exists", ss: compileBenchStylesheet(b, conditionalExistsStylesheet)},
		{name: "document", ss: compileBenchStylesheet(b, conditionalDocumentStylesheet)},
	}
	for _, form := range forms {
		b.Run(form.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := xslt3.Transform(b.Context(), source, form.ss); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

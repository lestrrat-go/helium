package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// A secondary (href) JSON result document must reject duplicate keys
// (SERE0022) when allow-duplicate-names is not "yes", exactly like the primary
// path. The final SerializeItems pass swallows serialization errors, so the
// duplicate-key check has to happen at the result-document commit point.
func TestResultDocumentSecondaryJSONDuplicateKeyRejected(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:output name="j" method="json" build-tree="no"/>
  <xsl:template match="/">
    <xsl:result-document href="out.json" format="j">
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.Error(t, err, "a secondary JSON result document with duplicate keys must fail")
	require.ErrorContains(t, err, "SERE0022")
}

// With allow-duplicate-names="yes" the same secondary JSON result document is
// accepted, confirming the new check honors the serialization parameter.
func TestResultDocumentSecondaryJSONDuplicateKeyAllowed(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:output name="j" method="json" build-tree="no"/>
  <xsl:template match="/">
    <xsl:result-document href="out.json" format="j" allow-duplicate-names="yes">
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.NoError(t, err,
		"allow-duplicate-names=yes must permit duplicate JSON keys in a secondary result document")
}

// A primary xsl:result-document that carries only AVT-only serialization
// attributes (media-type, html-version, include-content-type, etc.) must NOT
// force the output method to become explicit. When the base xsl:output did not
// explicitly set a method, html/xhtml auto-detection has to keep working: a
// result tree rooted at <html> in no namespace serializes with the HTML method
// (void elements like <br> are not self-closed).
func TestResultDocumentPrimaryAVTOnlyKeepsHTMLAutoDetect(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output encoding="UTF-8"/>
  <xsl:template match="/">
    <xsl:result-document media-type="{'text/html'}"><html><body><br/></body></html></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<br>",
		"AVT-only primary result-document must keep html auto-detection (void br not self-closed)")
	require.NotContains(t, out, "<br/>",
		"forcing MethodExplicit must not disable html auto-detection back to XML serialization")
}

// An invalid AVT value for ANY boolean serialization parameter on
// xsl:result-document must raise SEPM0016 rather than being silently coerced to
// false. All boolean serialization-param AVTs route through one shared helper,
// so a single invalid value is enough to fail each one.
func TestResultDocumentInvalidBooleanAVTRaisesSEPM0016(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr string
	}{
		{name: "indent", attr: `indent="{'bogus'}"`},
		{name: "byte-order-mark", attr: `byte-order-mark="{'bogus'}"`},
		{name: "include-content-type", attr: `include-content-type="{'bogus'}"`},
		{name: "escape-uri-attributes", attr: `escape-uri-attributes="{'bogus'}"`},
		{name: "omit-xml-declaration", attr: `omit-xml-declaration="{'bogus'}"`},
		{name: "undeclare-prefixes", attr: `undeclare-prefixes="{'bogus'}"`},
		{name: "build-tree", attr: `build-tree="{'bogus'}"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document `+tc.attr+`>
      <out/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

			_, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
			require.Error(t, err,
				"an invalid AVT value for a boolean serialization parameter must fail")
			require.ErrorContains(t, err, "SEPM0016")
		})
	}
}

// build-tree is an AVT, not a static compile-time bool. A build-tree AVT whose
// evaluation raises a dynamic error must surface that error instead of being
// silently ignored (which is what happened when build-tree was parsed only with
// parseXSDBool at compile time and dropped on a non-constant value).
func TestResultDocumentBuildTreeAVTEvaluated(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document build-tree="{1 idiv 0}">
      <out/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.Error(t, err,
		"a build-tree AVT that raises a dynamic error must not be silently ignored")
	require.ErrorContains(t, err, "FOAR0001")
}

// A primary xsl:result-document whose ONLY serialization attribute is
// suppress-indentation must still contribute that override. Before the fix the
// hasAny preflight gate omitted suppress-indentation, so evalResultDocOutputDef
// returned nil overrides and the attribute was silently dropped. With the base
// xsl:output indenting, suppress-indentation must keep the named element's
// children on a single line.
func TestResultDocumentSoleSuppressIndentationHonored(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output indent="yes"/>
  <xsl:template match="/">
    <xsl:result-document suppress-indentation="p">
      <doc><p><b>x</b><i>y</i></p></doc>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<p><b>x</b><i>y</i></p>",
		"sole suppress-indentation override must be honored (p children not indented)")
}

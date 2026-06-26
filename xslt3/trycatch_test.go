package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// Variables declared inside xsl:try (or its xsl:catch) must not leak into the
// surrounding scope. After the instruction completes, an outer variable of the
// same name must still resolve to its outer value.
func TestTryDoesNotLeakVariables(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			// Try body succeeds; inner $x must not shadow outer $x afterward.
			name: "success",
			body: `
      <xsl:try>
        <xsl:variable name="x" select="'inner'"/>
        <xsl:catch/>
      </xsl:try>`,
		},
		{
			// Try body fails; catch runs and declares $x, which must not leak.
			name: "catch",
			body: `
      <xsl:try>
        <xsl:sequence select="1 div xs:integer('not-a-number')"/>
        <xsl:catch>
          <xsl:variable name="x" select="'inner'"/>
        </xsl:catch>
      </xsl:try>`,
		},
		{
			// rollback-output="no" with a successful try body.
			name: "no-rollback-success",
			body: `
      <xsl:try rollback-output="no">
        <xsl:variable name="x" select="'inner'"/>
        <xsl:catch/>
      </xsl:try>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <xsl:variable name="x" select="'outer'"/>
    <out>`+tc.body+`<xsl:value-of select="$x"/></out>
  </xsl:template>
</xsl:stylesheet>`)

			result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Contains(t, result, ">outer<")
			require.NotContains(t, result, ">inner<")
		})
	}
}

// A tunnel parameter set by an xsl:call-template whose with-param evaluation
// later fails must not leak into templates invoked from the surrounding
// xsl:catch. The tunnel param is evaluated (mutating the active tunnel map)
// before a sibling with-param raises a dynamic error; the error is caught, and
// a second template called from xsl:catch must see the tunnel param as absent.
func TestCallTemplateTunnelParamDoesNotLeakAcrossCaughtError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:try>
        <xsl:call-template name="consume">
          <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
          <xsl:with-param name="bad" select="xs:integer('not-a-number')"/>
        </xsl:call-template>
        <xsl:catch>
          <xsl:call-template name="check"/>
        </xsl:catch>
      </xsl:try>
    </out>
  </xsl:template>

  <xsl:template name="consume">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="bad"/>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "NOLEAK")
	require.NotContains(t, result, "LEAKED")
}

// A tunnel parameter must not become observable to a template invoked from a
// LATER sibling with-param's BODY before control is actually transferred to the
// target template. Here the first with-param sets the tunnel param, and a later
// with-param body contains an xsl:try whose error is caught; the xsl:catch calls
// another template that reads the same tunnel param. Because the value is still
// being assembled (the call/next-match/apply-imports has not yet handed control
// to its target), the called template must see the tunnel param as ABSENT.
// This is the two-phase guarantee that xsl:apply-templates already provides.
func TestTunnelParamNotObservedByLaterWithParamBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{
			name: "call-template",
			src: `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:call-template name="consume">
        <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
        <xsl:with-param name="probe">
          <xsl:try>
            <xsl:sequence select="xs:integer('not-a-number')"/>
            <xsl:catch>
              <xsl:call-template name="check"/>
            </xsl:catch>
          </xsl:try>
        </xsl:with-param>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="consume">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="probe"/>
    <xsl:copy-of select="$probe"/>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`,
		},
		{
			name: "next-match",
			src: `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="root" priority="2">
    <out>
      <xsl:next-match>
        <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
        <xsl:with-param name="probe">
          <xsl:try>
            <xsl:sequence select="xs:integer('not-a-number')"/>
            <xsl:catch>
              <xsl:call-template name="check"/>
            </xsl:catch>
          </xsl:try>
        </xsl:with-param>
      </xsl:next-match>
    </out>
  </xsl:template>

  <xsl:template match="root" priority="1">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="probe"/>
    <xsl:copy-of select="$probe"/>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, tc.src)
			result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Contains(t, result, "NOLEAK")
			require.NotContains(t, result, "LEAKED")
		})
	}
}

// Same two-phase guarantee for xsl:apply-imports: a tunnel with-param set by an
// earlier sibling must not be visible to a template invoked from a later
// with-param body whose inner error is caught, before control is transferred to
// the imported template.
func TestApplyImportsTunnelParamNotObservedByLaterWithParamBody(t *testing.T) {
	const importedURI = "mem:/imported-tunnel.xsl"

	imported := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="probe"/>
    <xsl:copy-of select="$probe"/>
  </xsl:template>
</xsl:stylesheet>`

	main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import href="` + importedURI + `"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="root">
    <out>
      <xsl:apply-imports>
        <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
        <xsl:with-param name="probe">
          <xsl:try>
            <xsl:sequence select="xs:integer('not-a-number')"/>
            <xsl:catch>
              <xsl:call-template name="check"/>
            </xsl:catch>
          </xsl:try>
        </xsl:with-param>
      </xsl:apply-imports>
    </out>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`

	resolver := &memResolver{files: map[string]string{importedURI: imported}}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI("mem:/main.xsl").
		URIResolver(resolver).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte("<root/>"))
	require.NoError(t, err)

	out, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, out, "NOLEAK")
	require.NotContains(t, out, "LEAKED")
}

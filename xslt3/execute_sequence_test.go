package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestDocumentNodeContentWithDoctype checks a document node added to element
// content when the source document has a DOCTYPE declaration. The document
// node is replaced by its children (XSLT 3.0 §5.7.1), and XDM has no node
// kind for a DTD, so the DOCTYPE is not one of them: each path that splices a
// document node's children copies the element and the comment around the
// DOCTYPE, and nothing else.
func TestDocumentNodeContentWithDoctype(t *testing.T) {
	const srcXML = `<!--c--><!DOCTYPE r [<!ELEMENT r ANY>]><r/>`
	const want = `<out><!--c--><r/></out>`

	tests := []struct {
		name  string
		decls string
		body  string
	}{
		{name: "xsl:sequence", body: `<out><xsl:sequence select="/"/></out>`},
		{name: "xsl:try select", body: `<out><xsl:try select="/"><xsl:catch/></xsl:try></out>`},
		{name: "xsl:on-empty", body: `<out><xsl:sequence select="()"/><xsl:on-empty select="/"/></out>`},
		{
			name:  "template with as",
			decls: `<xsl:template name="t" as="node()*"><xsl:sequence select="/"/></xsl:template>`,
			body:  `<out><xsl:call-template name="t"/></out>`,
		},
		{
			name: "temporary tree",
			body: `<xsl:variable name="v"><out><xsl:sequence select="/"/></out></xsl:variable>` +
				`<xsl:copy-of select="$v"/>`,
		},
		{
			name: "xsl:where-populated",
			body: `<out><xsl:where-populated><xsl:document><xsl:copy-of select="/"/></xsl:document>` +
				`</xsl:where-populated></out>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			xsltSrc := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output omit-xml-declaration="yes"/>` + tc.decls + `
  <xsl:template match="/">` + tc.body + `</xsl:template>
</xsl:stylesheet>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(xsltSrc))
			require.NoError(t, err)
			ss, err := xslt3.CompileStylesheet(t.Context(), doc)
			require.NoError(t, err)
			src, err := helium.NewParser().Parse(t.Context(), []byte(srcXML))
			require.NoError(t, err)
			out, err := ss.Transform(src).Serialize(t.Context())
			require.NoError(t, err)
			require.Equal(t, want, strings.TrimSpace(out))
		})
	}
}

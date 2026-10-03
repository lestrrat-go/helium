package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAccumulatorXDMNodes checks that an accumulator visits the XDM nodes of
// a document only. A DOCTYPE declaration is not a child of the document node
// and an entity reference is not a child of its element, so a rule matching
// node() counts neither, nor the content of the entity.
func TestAccumulatorXDMNodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"doctype", `<?p1 x?><!--c1--><!DOCTYPE r [<!ELEMENT r ANY>]><!--c2--><r/><?p2 y?>`, "[0,5]"},
		{"entity reference", `<!DOCTYPE r [<!ENTITY e "<y/>">]><r><x/>&e;<z/></r>`, "[0,3]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			const xsltSrc = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:output method="text"/>
  <xsl:accumulator name="n" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="node()" select="$value + 1"/>
  </xsl:accumulator>
  <xsl:mode use-accumulators="n"/>
  <xsl:template match="/">[<xsl:value-of select="accumulator-before('n'), accumulator-after('n')"
    separator=","/>]</xsl:template>
</xsl:stylesheet>`
			require.Equal(t, tc.want, transformDoctype(t, xsltSrc, tc.src))
		})
	}
}

package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestAttributeSetCycleDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xsl  string
	}{
		{
			name: "direct self-cycle",
			xsl: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:attribute-set name="a" use-attribute-sets="a"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`,
		},
		{
			name: "indirect two-node cycle",
			xsl: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:attribute-set name="a" use-attribute-sets="b"/>
  <xsl:attribute-set name="b" use-attribute-sets="a"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, err := helium.Parse(ctx, []byte(tc.xsl))
			require.NoError(t, err)

			_, err = xslt3.CompileStylesheet(ctx, doc)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), "XTSE0720"),
				"expected XTSE0720 in error, got: %v", err)
		})
	}
}

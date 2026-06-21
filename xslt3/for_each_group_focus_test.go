package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestForEachGroupStartingWithPositionalPattern verifies that a positional
// pattern in group-starting-with sees the per-item focus (position/size of the
// population sequence) rather than the stale outer focus (ENG-005). The
// population is an atomic sequence, so the pattern predicate is evaluated with
// the item as context and reads ec.position/ec.size. With the bug, position()=3
// never matches (position stuck at the outer 1), producing a single group; the
// fix yields two groups split before the 3rd item.
func TestForEachGroupStartingWithPositionalPattern(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="(1,2,3,4,5)" group-starting-with=".[position()=3]">
        <group><xsl:value-of select="string-join(current-group()!string(.), ',')"/></group>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Equal(t, 2, strings.Count(out, "<group>"),
		"positional pattern should split into two groups, got: %s", out)
	require.Contains(t, out, "<group>1,2</group>")
	require.Contains(t, out, "<group>3,4,5</group>")
}

// TestForEachGroupEndingWithPositionalPattern verifies the same per-item focus
// handling for group-ending-with (ENG-005). position()=3 ends a group at the
// 3rd item of the population.
func TestForEachGroupEndingWithPositionalPattern(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="(1,2,3,4,5)" group-ending-with=".[position()=3]">
        <group><xsl:value-of select="string-join(current-group()!string(.), ',')"/></group>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Equal(t, 2, strings.Count(out, "<group>"),
		"positional pattern should split into two groups, got: %s", out)
	require.Contains(t, out, "<group>1,2,3</group>")
	require.Contains(t, out, "<group>4,5</group>")
}

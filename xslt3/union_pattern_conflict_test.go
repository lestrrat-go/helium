package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// ENG-007: an empty-body template with a union match pattern is split into
// separate rules sharing no Body[0] identity. When a single node matches more
// than one branch of the union (overlapping alternatives), the split branches
// must not be flagged as conflicting with each other under
// on-multiple-match="fail".
func TestUnionPatternEmptyBodyNoFalseConflict(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="root/*"/></out></xsl:template>
  <xsl:template match="node() | *"/>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(source).
		OnMultipleMatch(xslt3.OnMultipleMatchFail).
		Serialize(t.Context())
	require.NoError(t, err, "split union branches must not self-conflict")
	require.Contains(t, result, "<out/>")
}

// A genuine conflict between two DIFFERENT templates of equal precedence and
// priority must still raise XTDE0540 under on-multiple-match="fail".
func TestUnionPatternGenuineConflictStillFails(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="root/*"/></out></xsl:template>
  <xsl:template match="a" priority="1"><x/></xsl:template>
  <xsl:template match="a" priority="1"><y/></xsl:template>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).
		OnMultipleMatch(xslt3.OnMultipleMatchFail).
		Serialize(t.Context())
	require.Error(t, err, "genuine conflict must raise XTDE0540")
	require.Contains(t, err.Error(), "XTDE0540")
}

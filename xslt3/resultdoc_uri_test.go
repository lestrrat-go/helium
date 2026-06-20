package xslt3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A-006: duplicate xsl:result-document detection (XTDE1490) must key on the
// canonical (resolved) output URI, not the raw href. Two result documents whose
// hrefs resolve to the same absolute URI ("a/../out.xml" and "out.xml") under the
// same base output URI target the same document and must be rejected.
func TestResultDocumentDuplicateURICanonical(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="a/../out.xml"><a/></xsl:result-document>
    <xsl:result-document href="out.xml"><b/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		Do(t.Context())
	require.Error(t, err, "two result documents resolving to the same canonical URI must fail")
	require.Contains(t, err.Error(), "XTDE1490")
}

// A-007: an xsl:result-document format AVT that raises a dynamic error
// (e.g. {1 idiv 0}) must surface that error rather than silently falling back
// to the static/default format.
func TestResultDocumentFormatAVTErrorPropagates(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="out.xml" format="{1 idiv 0}"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.Error(t, err, "a dynamic error in the format AVT must not be swallowed")
	require.True(t, strings.Contains(err.Error(), "idiv") || strings.Contains(err.Error(), "FOAR0001") ||
		strings.Contains(err.Error(), "division") || strings.Contains(err.Error(), "zero"),
		"error should reflect the division-by-zero dynamic error, got: %v", err)
}

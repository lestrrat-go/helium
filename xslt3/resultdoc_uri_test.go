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

// A result-document whose format AVT raises a dynamic error inside xsl:try must
// be caught, and the URI it targeted must NOT remain reserved: a subsequent
// xsl:catch that writes the SAME href must succeed (no spurious XTDE1490),
// because no result document was ever committed to that URI.
func TestResultDocumentFormatAVTErrorReleasesURIInTryCatch(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document href="out.xml" format="{1 idiv 0}"><a/></xsl:result-document>
      <xsl:catch>
        <xsl:result-document href="out.xml"><b/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err, "the caught result-document must release its URI reservation so the catch can reuse the same href")
}

// A relative secondary href ("out.xml") and the equivalent absolute href
// ("file:///base/dir/out.xml") under the same base output URI denote the SAME
// file. The XTDE1490 duplicate-detection key must canonicalize URI-wise,
// PRESERVING the file: scheme for BOTH forms, so the two collide. (Regression:
// helium.BuildURI strips the file: scheme for file: bases, turning the relative
// href into "/base/dir/out.xml" while the absolute href stayed
// "file:///base/dir/out.xml", so the duplicate was missed.)
func TestResultDocumentDuplicateRelativeVsAbsoluteFileURI(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="out.xml"><a/></xsl:result-document>
    <xsl:result-document href="file:///base/dir/out.xml"><b/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		Do(t.Context())
	require.Error(t, err, "relative and absolute file: hrefs denoting the same file must collide")
	require.Contains(t, err.Error(), "XTDE1490")
}

// A primary xsl:result-document whose serialization parameter AVT raises a
// dynamic error must fail BEFORE any primary output is emitted. When wrapped in
// xsl:try, the released URI reservation lets an xsl:catch write the primary
// result document — but the failed instruction must not have left partial
// primary content behind, so the catch's document is the SOLE primary output
// (no double-primary "<a/><b/>").
func TestResultDocumentPrimarySerializationAVTErrorNoDoublePrimary(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document standalone="{1 idiv 0}"><a/></xsl:result-document>
      <xsl:catch>
        <xsl:result-document><b/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err, "the caught primary result-document must succeed without a spurious conflict")
	require.Contains(t, out, "<b/>", "the catch's primary result document must be emitted")
	require.NotContains(t, out, "<a/>", "the failed primary result document must not leave partial output behind")
}

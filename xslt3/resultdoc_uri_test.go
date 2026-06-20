package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
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

// A primary xsl:result-document whose body EMITS content ("<a/>") and THEN
// throws (xsl:message terminate="yes") inside xsl:try must not leave that
// partial content in the primary tree. The deferred release of the "" URI
// reservation lets the xsl:catch write a fresh primary result document, so the
// catch's "<b/>" must be the SOLE primary output — never "<a/><b/>". This is the
// double-primary regression the buffered direct-write path prevents: the
// body's content is staged in a temporary frame and only spliced into the
// primary tree after the body succeeds.
func TestResultDocumentPrimaryThrowingBodyNoDoublePrimary(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document>
        <a/>
        <xsl:message terminate="yes">boom</xsl:message>
      </xsl:result-document>
      <xsl:catch>
        <xsl:result-document><b/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err, "the caught primary result-document must succeed without a spurious conflict")
	require.Contains(t, out, "<b/>", "the catch's primary result document must be emitted")
	require.NotContains(t, out, "<a/>", "the thrown body must not leave partial primary output behind")
}

// A SECONDARY xsl:result-document whose serialization parameter AVT raises a
// dynamic error (method="{1 idiv 0}") must fail in a PREFLIGHT, BEFORE its body
// executes — symmetric with the primary path. When wrapped in xsl:try and
// caught, the body (which itself writes a NESTED result document) must never run,
// so the handler must NOT receive the nested result document: the failed outer
// instruction's transaction rolls back with no body executed and no nested
// commit. (Regression: the secondary path evaluated serialization AVTs AFTER the
// body, so the nested result document committed before the outer AVT error
// surfaced, leaking a stale nested document into the caught state.)
func TestResultDocumentSecondarySerializationAVTErrorNoNestedCommit(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document href="outer.xml" method="{1 idiv 0}">
        <xsl:result-document href="nested.xml"><nested/></xsl:result-document>
      </xsl:result-document>
      <xsl:catch>
        <xsl:result-document href="caught.xml"><caught/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.NoError(t, err,
		"the caught secondary result-document must release its URI and the catch must succeed")

	_, gotNested := collector.docs["nested.xml"]
	require.False(t, gotNested,
		"the outer serialization AVT must fail in a preflight before the body runs; no nested result document may commit")
	_, gotCaught := collector.docs["caught.xml"]
	require.True(t, gotCaught, "the catch's result document must be delivered")
}

// A-007 (PR #649 round 7): the PRIMARY xsl:result-document branches for
// validation="strict|lax" previously RETURNED before the serialization-AVT
// preflight, so a failing serialization AVT (standalone="{1 idiv 0}") was
// silently swallowed and the instruction returned <a/> with err=nil. The
// preflight now runs above the validation= return, so the dynamic error is
// surfaced and (here) catchable in xsl:try, leaving the catch's <b/> as the
// sole primary output with no partial <a/>.
func TestResultDocumentPrimaryValidationStrictSerializationAVTError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document validation="strict" standalone="{1 idiv 0}"><a/></xsl:result-document>
      <xsl:catch>
        <xsl:result-document><b/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err, "the caught validation=strict primary result-document must succeed")
	require.Contains(t, out, "<b/>", "the catch's primary result document must be emitted")
	require.NotContains(t, out, "<a/>", "the failed validation=strict primary must not leave partial output behind")
}

// A-007 (PR #649 round 7): the same swallow existed in the primary
// type="..." branch — it returned before the serialization-AVT preflight. The
// hoisted preflight surfaces the failing AVT, catchable in xsl:try.
func TestResultDocumentPrimaryTypeSerializationAVTError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document type="xs:untyped" standalone="{1 idiv 0}"><a/></xsl:result-document>
      <xsl:catch>
        <xsl:result-document><b/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err, "the caught type=... primary result-document must succeed")
	require.Contains(t, out, "<b/>", "the catch's primary result document must be emitted")
	require.NotContains(t, out, "<a/>", "the failed type=... primary must not leave partial output behind")
}

// A-007 (PR #649 round 7): a primary xsl:result-document with validation="strict"
// and VALID serialization AVTs must still apply its overrides. Pre-fix the
// validation= branch returned before committing primaryOutputOverrides, so the
// standalone="yes" override was dropped from the effective primary output def.
func TestResultDocumentPrimaryValidationStrictAppliesOverrides(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document validation="strict" standalone="yes"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	_, err := inv.Do(t.Context())
	require.NoError(t, err)
	od := inv.ResolvedOutputDef()
	require.NotNil(t, od, "resolved output def must be populated after Do")
	require.Equal(t, "yes", od.Standalone,
		"the validation=strict primary result-document's standalone override must reach the effective output def")
}

// XTDE1490 duplicate detection must collapse dot-segments in ABSOLUTE hrefs.
// "file:///base/dir/a/../out.xml" and "file:///base/dir/out.xml" denote the same
// file and must collide. (Regression: absolute hrefs were keyed without
// dot-segment normalization, so the "a/.." form did not collide with the plain
// form.)
func TestResultDocumentDuplicateAbsoluteDotSegments(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="file:///base/dir/a/../out.xml"><a/></xsl:result-document>
    <xsl:result-document href="file:///base/dir/out.xml"><b/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		Do(t.Context())
	require.Error(t, err, "two absolute file: hrefs denoting the same file (after dot-segment collapse) must collide")
	require.Contains(t, err.Error(), "XTDE1490")
}

// Inside a secondary result-document, a NESTED secondary result-document that
// targets a relative href ("inner.xml") and another that targets the equivalent
// absolute file: href ("file:///base/dir/inner.xml") denote the SAME file and
// must collide with XTDE1490. This requires the enclosing secondary output to
// update current-output-uri() with a scheme-preserving (canonical) URI so the
// nested relative href resolves to the same key as its absolute equivalent.
// (Regression: helium.BuildURI strips the file: scheme for file: bases, so the
// nested relative href keyed as "/base/dir/inner.xml" while the absolute href
// stayed "file:///base/dir/inner.xml", missing the duplicate.)
func TestResultDocumentNestedDuplicateRelativeVsAbsoluteFileURI(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="outer.xml">
      <outer>
        <xsl:result-document href="inner.xml"><a/></xsl:result-document>
        <xsl:result-document href="file:///base/dir/inner.xml"><b/></xsl:result-document>
      </outer>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		Do(t.Context())
	require.Error(t, err, "nested relative and absolute file: hrefs denoting the same file must collide")
	require.Contains(t, err.Error(), "XTDE1490")
}

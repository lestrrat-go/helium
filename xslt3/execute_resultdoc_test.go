package xslt3_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
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
// (e.g. {1 idiv 0}) must surface that error, falling back to no
// static/default format.
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

// ENG-006: per XSLT 3.0 §26.2 a secondary result document's base URI is its href
// resolved against the BASE OUTPUT URI, NOT against the stylesheet's base URI. So
// base-uri()/Document.URL() on the delivered secondary tree must reflect the base
// output URI. (Regression: the secondary path set the document URL by resolving
// href against ec.stylesheet.baseURI, yielding the stylesheet dir
// "file:///style/dir/secondary.xml" instead of the output dir
// "file:///out/secondary.xml".)
func TestResultDocumentSecondaryBaseURIFromOutputURI(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="secondary.xml"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`))
	require.NoError(t, err)

	// Compile with a stylesheet base URI that DIFFERS from the base output URI so
	// a wrong (stylesheet-relative) resolution is distinguishable from the correct
	// (output-relative) one.
	ss, err := xslt3.NewCompiler().
		BaseURI("file:///style/dir/main.xsl").
		Compile(t.Context(), doc)
	require.NoError(t, err)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err = ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///out/main.xml").
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.NoError(t, err)

	got, ok := collector.docs["secondary.xml"]
	require.True(t, ok, "the secondary result document must be delivered")
	require.Equal(t, "file:///out/secondary.xml", got.URL(),
		"a secondary result document's base URI must be its href resolved against the base output URI, not the stylesheet base URI")
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

// XSLT3-ADV-001: every error-prone serialization AVT on xsl:result-document must
// be evaluated in the preflight, even when it is the ONLY serialization attribute
// present. Previously media-type, html-version, include-content-type,
// allow-duplicate-names and output-version were absent from the preflight
// `hasAny` gate (or never evaluated), so when one of them was the sole
// serialization attribute the gate short-circuited and a failing AVT (e.g.
// {1 idiv 0}) was silently swallowed: the body emitted output with err=nil.
func TestResultDocumentSerializationAVTErrorPropagates(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr string
	}{
		{"media-type", `media-type="{1 idiv 0}"`},
		{"output-version", `output-version="{1 idiv 0}"`},
		{"html-version", `html-version="{1 idiv 0}"`},
		{"include-content-type", `include-content-type="{1 idiv 0}"`},
		{"allow-duplicate-names", `allow-duplicate-names="{1 idiv 0}"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document `+tc.attr+`><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

			_, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
			require.Error(t, err, "a dynamic error in the %s AVT must not be swallowed", tc.name)
			require.True(t, strings.Contains(err.Error(), "idiv") || strings.Contains(err.Error(), "FOAR0001") ||
				strings.Contains(err.Error(), "division") || strings.Contains(err.Error(), "zero"),
				"error should reflect the division-by-zero dynamic error, got: %v", err)
		})
	}
}

// XSLT3-ADV-001: a primary xsl:result-document whose serialization AVT raises a
// dynamic error must fail in the preflight, BEFORE any primary output is emitted,
// so an enclosing xsl:catch can write the sole primary result document with no
// partial <a/> left behind. This must hold for the AVTs that were previously
// missing from the preflight gate (media-type / output-version shown here).
func TestResultDocumentSerializationAVTErrorObservableInTryCatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr string
	}{
		{"media-type", `media-type="{1 idiv 0}"`},
		{"output-version", `output-version="{1 idiv 0}"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document `+tc.attr+`><a/></xsl:result-document>
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
		})
	}
}

// XSLT3-ADV-001: output-version supplied as a valid AVT on a primary
// xsl:result-document must reach the effective output definition. Pre-fix the
// output-version AVT was never evaluated/applied, so the override was dropped.
func TestResultDocumentPrimaryOutputVersionAVTApplied(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document method="xml" output-version="{concat('1','.','1')}"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	_, err := inv.Do(t.Context())
	require.NoError(t, err)
	od := inv.ResolvedOutputDef()
	require.NotNil(t, od, "resolved output def must be populated after Do")
	require.Equal(t, "1.1", od.Version,
		"the primary result-document's output-version AVT override must reach the effective output def")
}

// XSLT3-ADV-001: with a default method="json" output, a primary
// xsl:result-document allow-duplicate-names="{...}" AVT that resolves to true
// must reach the transform-level SERE0022 dup-key validation. Pre-fix the
// final validation read ss.outputs[""] (default, no) and the override merge
// never copied AllowDuplicateNames, so duplicate JSON keys were wrongly
// rejected even though the result-document permitted them.
func TestResultDocumentPrimaryJSONAllowDuplicateNamesAVT(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="json"/>
  <xsl:template match="/">
    <xsl:result-document allow-duplicate-names="{true()}">
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	_, err := inv.Do(t.Context())
	require.NoError(t, err,
		"duplicate JSON keys must be accepted when the primary result-document's allow-duplicate-names AVT resolves to true")
	od := inv.ResolvedOutputDef()
	require.NotNil(t, od)
	require.True(t, od.AllowDuplicateNames,
		"the primary result-document allow-duplicate-names override must reach the effective output def")
}

// A bare primary <xsl:result-document> (no serialization attributes of its own)
// must still honor a stylesheet-level
// <xsl:output method="json" allow-duplicate-names="yes"/>. Pre-fix the JSON
// dup-key check hard-coded allowDupes=false whenever the result-document carried
// no overrides (primaryOverrides==nil), so duplicate keys were wrongly rejected
// even though the default output definition permitted them.
func TestResultDocumentPrimaryJSONAllowDuplicateNamesDefaultOutput(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="json" allow-duplicate-names="yes"/>
  <xsl:template match="/">
    <xsl:result-document>
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	_, err := inv.Do(t.Context())
	require.NoError(t, err,
		"duplicate JSON keys must be accepted when the default xsl:output sets allow-duplicate-names=yes, even for a bare result-document")
}

// Secondary equivalent of TestResultDocumentPrimaryJSONAllowDuplicateNamesDefaultOutput:
// a bare SECONDARY xsl:result-document (href, no format, no serialization
// attributes of its own) must still honor a stylesheet-level
// <xsl:output method="json" allow-duplicate-names="yes"/>. Pre-fix
// buildEffectiveOutputDef left base empty when there was no named format and no
// parameter-document, so the SERE0022 dup-key check saw allow-duplicate-names=false
// and wrongly rejected duplicate keys the default output permitted.
func TestResultDocumentSecondaryJSONAllowDuplicateNamesDefaultOutput(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:output method="json" allow-duplicate-names="yes"/>
  <xsl:template match="/">
    <xsl:result-document href="out.json">
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.NoError(t, err,
		"duplicate JSON keys must be accepted when the default xsl:output sets allow-duplicate-names=yes, even for a bare secondary result-document")
}

// A primary xsl:result-document undeclare-prefixes="{...}" AVT that resolves to
// true must reach the effective output definition. Pre-fix undeclare-prefixes was
// validated as a boolean serialization attribute at compile time but never
// compiled into an AVT, never evaluated, and never copied into the merged output
// def, so a dynamic value was silently ignored.
func TestResultDocumentPrimaryUndeclarePrefixesAVT(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document undeclare-prefixes="{true()}">
      <out/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	_, err := inv.Do(t.Context())
	require.NoError(t, err,
		"a primary result-document undeclare-prefixes AVT resolving to true must not fail")
	od := inv.ResolvedOutputDef()
	require.NotNil(t, od)
	require.True(t, od.UndeclarePrefixes,
		"the primary result-document undeclare-prefixes override must reach the effective output def")
}

// XSLT3-101: a SECONDARY xsl:result-document whose href resolves to the PRINCIPAL
// (base) output URI denotes the same final result tree as the principal output and
// must collide with XTDE1490. The principal output URI is seeded into the used-URI
// set so a secondary href resolving to it (here href="main.xml" against
// BaseOutputURI="file:///base/dir/main.xml") is detected. (Regression: the primary
// output keyed only on the "" sentinel and nothing reserved the canonical principal
// URI, so a secondary resolving to it evaded the duplicate-URI check and silently
// produced two final results for the same URI.)
func TestResultDocumentSecondaryResolvingToPrincipalURICollides(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="main.xml"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		Do(t.Context())
	require.Error(t, err, "a secondary href resolving to the principal output URI must collide")
	require.Contains(t, err.Error(), "XTDE1490")
}

// XSLT3-101: an absolute secondary href equal to the principal (base) output URI
// must collide with XTDE1490, exactly like the relative form above.
func TestResultDocumentSecondaryAbsoluteEqualToPrincipalURICollides(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="file:///base/dir/main.xml"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		Do(t.Context())
	require.Error(t, err, "an absolute secondary href equal to the principal output URI must collide")
	require.Contains(t, err.Error(), "XTDE1490")
}

// XSLT3-101: a secondary xsl:result-document whose href resolves to a DIFFERENT URI
// than the principal (base) output URI must still succeed — seeding the principal
// URI must not reject distinct secondary destinations.
func TestResultDocumentSecondaryDistinctFromPrincipalURISucceeds(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="other.xml"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///base/dir/main.xml").
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.NoError(t, err, "a secondary href distinct from the principal output URI must succeed")
	_, ok := collector.docs["other.xml"]
	require.True(t, ok, "the distinct secondary result document must be delivered")
}

// An xsl:result-document with a CONSTANT (non-AVT) invalid allow-duplicate-names
// value must be rejected at compile time with SEPM0016, like every other boolean
// serialization attribute. (Regression: allow-duplicate-names was missing from
// the compile-time boolean-validation list, so "bogus" compiled cleanly.)
func TestResultDocumentAllowDuplicateNamesInvalidConstantCompileError(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document allow-duplicate-names="bogus"><a/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`))
	require.NoError(t, err)

	_, err = xslt3.CompileStylesheet(t.Context(), doc)
	require.Error(t, err, "an invalid constant allow-duplicate-names must fail compilation")
	require.Contains(t, err.Error(), "SEPM0016")
}

// An xsl:result-document whose allow-duplicate-names AVT resolves to an invalid
// xs:boolean lexical form must raise a dynamic SEPM0016 error, NOT silently fall
// back to an inherited value. (Regression: an invalid AVT result was dropped on
// the floor, leaving an inherited allow-duplicate-names="yes" wrongly in force
// and permitting duplicate JSON keys.)
func TestResultDocumentAllowDuplicateNamesInvalidAVTDynamicError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="json" allow-duplicate-names="yes"/>
  <xsl:template match="/">
    <xsl:result-document allow-duplicate-names="{'bogus'}">
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	_, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.Error(t, err,
		"an invalid allow-duplicate-names AVT must raise a dynamic error, not fall back to the inherited value")
	require.Contains(t, err.Error(), "SEPM0016")
}

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
// xsl:result-document must raise SEPM0016, and is never silently coerced to
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

// XHTML/HTML5 serialization defaults omit-xml-declaration to "yes" only when the
// value was NOT explicitly set. A primary xsl:result-document that evaluates
// omit-xml-declaration="{false()}" explicitly specifies it, so the XML
// declaration must be emitted. Before the fix the OmitDeclarationExplicit flag
// was dropped on the result-document AVT path and the html5 serializer flipped
// the declaration back off.
func TestResultDocumentXHTMLOmitDeclarationFalseExplicit(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xhtml" html-version="5"/>
  <xsl:template match="/">
    <xsl:result-document omit-xml-declaration="{false()}">
      <html xmlns="http://www.w3.org/1999/xhtml"><body><p>x</p></body></html>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<?xml",
		"explicit omit-xml-declaration=false on the result-document must keep the XML declaration in XHTML/HTML5 output")
}

// Finding 1 (PR #649 round 4): the buffered primary direct-write path must
// preserve the REAL base frame's capture state. When the default output method
// is json/adaptive (an item-serialization method) the base frame has
// captureItems=true, so atomic values produced by xsl:sequence inside a primary
// xsl:result-document MUST be preserved as separate XDM items, and never
// stringified into the DOM as text. With the regression, the buffer frame was
// created without captureItems/sequenceMode, so the atomics were written as a
// single merged text node instead of three integer items.
func TestResultDocumentPrimaryAdaptivePreservesAtomicItems(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:result-document>
      <xsl:sequence select="(1, 2, 3)"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	capture := &primaryItemsCapture{}
	_, err := ss.Transform(parseTransformSource(t)).
		PrimaryItemsHandler(capture).
		Do(t.Context())
	require.NoError(t, err)

	require.NotNil(t, capture.seq, "expected primary items to be captured")
	require.Equal(t, 3, capture.seq.Len(),
		"three atomic values must be preserved as three XDM items, not stringified into a single text node")
	for i, want := range []string{"1", "2", "3"} {
		av, ok := capture.seq.Get(i).(xpath3.AtomicValue)
		require.True(t, ok, "captured item %d must be an atomic value, not a node/text item", i)
		s, sErr := xpath3.AtomicToString(av)
		require.NoError(t, sErr)
		require.Equal(t, want, s, "captured atomic %d must retain its integer value", i)
	}
}

// Finding 2 (PR #649 round 4): per-href result-document state is a transaction
// with a single commit point. The failed attempt below uses method="adaptive"
// (an item-serialization method), so its body populates resultDocItems[href]
// BEFORE post-body validation. validation="strict" over a two-root document then
// fails (XTTE1550 from validateDocumentStructure). Pre-fix, resultDocItems and
// resultDocOutputDefs were mutated before the validation step and before
// committed=true, so the caught failure left stale entries. The xsl:catch then
// writes the SAME href with a plain XML body (no items): the stale json items
// from the rolled-back attempt would be serialized into the catch's document by
// the end-of-transform materialization loop, contaminating it. The transaction
// fix stages all per-href state and publishes it only at the commit point, so
// the catch's <good/> is the SOLE content for that href.
func TestResultDocumentSecondaryValidationFailRollbackTryCatch(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output name="json" method="adaptive"/>
  <xsl:template match="/">
    <xsl:try>
      <xsl:result-document href="out.xml" format="json" validation="strict">
        <bad/>
        <alsobad/>
      </xsl:result-document>
      <xsl:catch>
        <xsl:result-document href="out.xml"><good/></xsl:result-document>
      </xsl:catch>
    </xsl:try>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		ResultDocumentHandler(collector).
		Do(t.Context())
	require.NoError(t, err,
		"the caught secondary result-document must release its URI so the catch reuses the same href")

	doc, ok := collector.docs["out.xml"]
	require.True(t, ok, "the catch must have delivered a result document for out.xml")
	root := findResultRoot(doc)
	require.NotNil(t, root, "delivered result document must have a root element")
	require.Equal(t, "good", root.Name(),
		"only the catch's <good/> may be delivered; the rolled-back attempt must leave no stale state")

	// The stale json items from the failed attempt must NOT have been serialized
	// into the catch's document: it must contain exactly one child, the <good/>
	// element, with no leftover text node from the rolled-back <bad/>/<alsobad/>.
	var childCount int
	for child := range helium.Children(doc) {
		childCount++
		require.Equal(t, helium.ElementNode, child.Type(),
			"no stale serialized text node may be appended to the catch's document")
	}
	require.Equal(t, 1, childCount, "the catch's document must contain only <good/>")
}

// A secondary xsl:result-document with an item-serialization method (adaptive)
// stages its element items into resultDocItems and serializes them in the
// end-of-transform materialization loop. When that serialization fails — here a
// two-element sequence whose text carries U+0001, an XML-invalid character — the
// error MUST surface from the transform, and must never be swallowed into a
// silently empty secondary document.
func TestResultDocumentSecondarySerializationErrorSurfaces(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="e" as="element()">
      <r><xsl:value-of select="codepoints-to-string(1)"/></r>
    </xsl:variable>
    <xsl:result-document href="out.txt" method="adaptive">
      <xsl:sequence select="($e, $e)"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	collector := &resultDocCollect{docs: map[string]*helium.Document{}}
	_, err := ss.Transform(parseTransformSource(t)).
		ResultDocumentHandler(collector).
		Do(t.Context())
	requireSERE0006(t, err)
}

type resultDocCollect struct {
	docs map[string]*helium.Document
}

func (c *resultDocCollect) HandleResultDocument(href string, doc *helium.Document, _ *xslt3.OutputDef) error {
	c.docs[href] = doc
	return nil
}

func findResultRoot(doc *helium.Document) helium.Node {
	for child := range helium.Children(doc) {
		if child.Type() == helium.ElementNode {
			return child
		}
	}
	return nil
}

func TestPrimaryResultDocumentAdaptiveMarkupTemporaryFrames(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		attributes string
	}{
		{
			name:       "ValidationLax",
			attributes: `validation="lax" method="adaptive" item-separator=" | "`,
		},
		{
			name:       "TypeNamedFormat",
			output:     `<xsl:output name="adaptive" method="adaptive" item-separator=" | "/>`,
			attributes: `type="xs:untyped" format="adaptive"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  `+tt.output+`
  <xsl:template match="/">
    <xsl:result-document `+tt.attributes+`>
      <xsl:comment select="'first'"/>
      <xsl:processing-instruction name="target" select="'data'"/>
      <xsl:comment select="'last'"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

			out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Equal(t, "<!--first--> | <?target data?> | <!--last-->", out)
			require.NotContains(t, out, "<?xml")
		})
	}
}

func TestPrimaryResultDocumentAdaptiveMarkupTemporaryFramesInvalidCharacter(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		attributes string
	}{
		{
			name:       "ValidationLax",
			attributes: `validation="lax" method="adaptive" item-separator=" | "`,
		},
		{
			name:       "TypeNamedFormat",
			output:     `<xsl:output name="adaptive" method="adaptive" item-separator=" | "/>`,
			attributes: `type="xs:untyped" format="adaptive"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  `+tt.output+`
  <xsl:template match="/">
    <xsl:result-document `+tt.attributes+`>
      <xsl:comment select="'first'"/>
      <xsl:processing-instruction name="target" select="codepoints-to-string(1)"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

			_, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			requireSERE0006(t, err)
		})
	}
}

// TestPrimaryResultDocumentUnlinksChildren guards against a tree-corruption bug
// where a primary xsl:result-document (validation="strict") moved children from
// a temporary document into the primary document via AddChild without unlinking
// them first. The temporary document's sibling links remained attached, which
// corrupted the result tree (and could hang sibling traversal) when the body
// produced a comment followed by the root element.
func TestPrimaryResultDocumentUnlinksChildren(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="" validation="strict">
      <xsl:comment>lead</xsl:comment>
      <out>body</out>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)

	done := make(chan struct{})
	var result string
	var err error
	go func() {
		result, err = ss.Transform(source).Serialize(t.Context())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Serialize hung: primary result-document corrupted the sibling chain")
	}

	require.NoError(t, err)
	require.Contains(t, result, "<!--lead-->")
	require.Contains(t, result, "<out>body</out>")
	require.Less(t, strings.Index(result, "<!--lead-->"), strings.Index(result, "<out>body</out>"),
		"comment must precede the root element")
}

// A primary xsl:result-document whose parameter-document OMITS a serialization
// parameter must inherit that parameter's value from the unnamed default
// xsl:output, not silently reset it to the Go zero value. Before the fix
// evalResultDocOutputDef took the parameter-document OutputDef as the WHOLE base
// when a parameter-document was present, dropping the default xsl:output: an
// omitted plain boolean (indent, byte-order-mark, allow-duplicate-names,
// undeclare-prefixes) then overwrote an inherited true with false.
func TestResultDocParamDocInheritsDefaultOutput(t *testing.T) {
	const paramDocURI = "http://example.invalid/params.xml"

	// Serialization parameter document that deliberately OMITS indent and
	// allow-duplicate-names; it only sets an unrelated parameter (encoding).
	const paramDoc = `<serialization-parameters xmlns="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<encoding value="utf-8"/>` +
		`</serialization-parameters>`

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != paramDocURI {
			return nil, errors.New("not found: " + uri)
		}
		return io.NopCloser(strings.NewReader(paramDoc)), nil
	})

	t.Run("indent inherited", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output indent="yes"/>
  <xsl:template match="/">
    <xsl:result-document parameter-document="{'`+paramDocURI+`'}">
      <root><child>x</child></root>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

		out, err := ss.Transform(parseTransformSource(t)).
			URIResolver(resolver).
			Serialize(t.Context())
		require.NoError(t, err)
		// indent="yes" inherited from the default xsl:output must still pretty-print
		// the nested element across lines ("<root>\n  <child>..."); a clobbered
		// indent=false would emit "<root><child>x</child></root>" on one line.
		require.Contains(t, out, "<root>\n", "inherited indent=yes must survive a parameter-document that omits indent")
	})

	t.Run("allow-duplicate-names inherited", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="json" allow-duplicate-names="yes"/>
  <xsl:template match="/">
    <xsl:result-document parameter-document="{'`+paramDocURI+`'}">
      <xsl:sequence select="map{1:'a','1':'b'}"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

		// The map's integer key 1 and string key "1" both become the JSON name
		// "1": a duplicate name that SERE0022 rejects unless allow-duplicate-names
		// ="yes". That value is set only on the default xsl:output and OMITTED by
		// the parameter-document, so it must be inherited, and never reset to false.
		inv := ss.Transform(parseTransformSource(t)).URIResolver(resolver)
		_, err := inv.Do(t.Context())
		require.NoError(t, err, "inherited allow-duplicate-names=yes must survive a parameter-document that omits it (no SERE0022)")
		od := inv.ResolvedOutputDef()
		require.NotNil(t, od)
		require.True(t, od.AllowDuplicateNames,
			"the default xsl:output allow-duplicate-names=yes must survive a parameter-document that omits it")
	})
}

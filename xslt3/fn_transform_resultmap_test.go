package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// wantTrue is the string value of an XPath expression that evaluates to true().
const wantTrue = "true"

// xslXMLVars binds the $xsl (stylesheet) and $xml (source) let-variables read by
// the fn:transform result-map test expressions.
func xslXMLVars(xsl, xml string) map[string]xpath3.Sequence {
	return map[string]xpath3.Sequence{
		"xsl": xpath3.SingleString(xsl),
		"xml": xpath3.SingleString(xml),
	}
}

// transformBool compiles and evaluates a boolean-valued XPath expression that
// drives the standalone fn:transform, returning its string value ("true" /
// "false"). vars supplies the let-bindings the expression reads.
func transformBool(t *testing.T, expr string, vars map[string]xpath3.Sequence) string {
	t.Helper()
	got, err := evalTransform(t, expr, nil, vars, transformFns())
	require.NoError(t, err)
	return got
}

// multiResultDocXSL emits one xsl:result-document per section and no principal
// output — the shape that must yield no principal ("output") entry (bug 30209).
const multiResultDocXSL = `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'><xsl:for-each select='//section'><xsl:result-document href='section{position()}.html'><out><xsl:value-of select='position()'/></out></xsl:result-document></xsl:for-each></xsl:template>
</xsl:stylesheet>`

const multiResultDocXML = `<doc><section>s1</section><section>s2</section><section>s3</section></doc>`

// TestFnTransformResultMapKeying exercises the fn:transform result-map assembly
// rules of F&O 3.1 §14.8.3: the principal-result key, the omission of a
// principal entry when only secondary result documents are produced, and the
// resolution of secondary result-document keys against the base output URI.
func TestFnTransformResultMapKeying(t *testing.T) {
	base := "http://www.w3.org/fots/fn/transform/output-doc.xml"
	multiVars := xslXMLVars(multiResultDocXSL, multiResultDocXML)

	testcases := []struct {
		name string
		expr string
	}{
		{
			// fn-transform-13 / 33 / 44: only secondary result documents, so the
			// original map has exactly the three secondary entries — no principal
			// entry under "output" nor under the base output URI.
			name: "no-principal-entry-when-only-secondary-docs",
			expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"base-output-uri":"` + base + `"})
			return map:size($r)=3 and not(map:contains($r,"output")) and not(map:contains($r,"` + base + `"))`,
		},
		{
			// fn-transform-13a / 37: secondary keys are the href resolved against
			// base-output-uri (an absolute URI), not the relative href as written.
			name: "secondary-keys-resolved-absolute",
			expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"base-output-uri":"` + base + `"})
			return contains(string-join(map:keys($r)),"www.w3.org/fots/fn/transform/section2.html")`,
		},
		{
			// fn-transform-33: same, serialized delivery. Assert on the original
			// map so a stray principal entry keyed by base-output-uri would fail.
			name: "serialized-no-principal-entry",
			expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"base-output-uri":"` + base + `","delivery-format":"serialized"})
			return map:size($r)=3 and not(map:contains($r,"output")) and not(map:contains($r,"` + base + `")) and contains(string-join(map:keys($r)),"section2")`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, wantTrue, transformBool(t, tc.expr, multiVars))
		})
	}
}

// principalOnlyXSL produces a single principal result element and no secondary
// result documents.
const principalOnlyXSL = `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'><out><xsl:value-of select='name(*)'/></out></xsl:template>
</xsl:stylesheet>`

// TestFnTransformPrincipalKey verifies that the principal result is keyed by the
// base output URI when one is supplied (fn-transform-16 / 17 / 35 / 45 / 88),
// and by "output" otherwise.
func TestFnTransformPrincipalKey(t *testing.T) {
	base := "http://www.w3.org/fots/fn/transform/output-doc.xml"
	vars := xslXMLVars(principalOnlyXSL, `<doc/>`)

	testcases := []struct {
		name string
		expr string
	}{
		{
			// fn-transform-17: principal keyed by base-output-uri, not "output".
			name: "document-principal-keyed-by-base-output-uri",
			expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"delivery-format":"document","base-output-uri":"` + base + `"})
			return not(map:contains($r,"output")) and map:contains($r,"` + base + `") and $r("` + base + `") instance of node()`,
		},
		{
			// fn-transform-35 / 45: serialized principal keyed by base-output-uri.
			name: "serialized-principal-keyed-by-base-output-uri",
			expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"delivery-format":"serialized","base-output-uri":"` + base + `"})
			return map:size($r)=1 and not(map:contains($r,"output")) and $r("` + base + `") instance of xs:string`,
		},
		{
			// No base-output-uri: principal keyed by the literal "output".
			name: "principal-keyed-by-output-without-base-output-uri",
			expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"delivery-format":"document"})
			return map:contains($r,"output") and $r("output") instance of node()`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, wantTrue, transformBool(t, tc.expr, vars))
		})
	}
}

// TestFnTransformRawPrincipalKeyedByBaseOutputURI mirrors fn-transform-88: raw
// delivery with a base-output-uri keys the principal result by that URI.
func TestFnTransformRawPrincipalKeyedByBaseOutputURI(t *testing.T) {
	xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0' default-mode='x'>
<xsl:template match='/' mode='#unnamed'>WRONG</xsl:template>
<xsl:template match='/' mode='x'>RIGHT</xsl:template>
</xsl:stylesheet>`
	expr := `let $r := transform(map{"stylesheet-text":$xsl,"delivery-format":"raw","base-output-uri":"http://example.com/","source-node":parse-xml('<a><b>89</b></a>')})
	return string($r("http://example.com/")) = 'RIGHT'`
	require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
}

// globalContextXSL is the fn-transform-82 variable-with-context stylesheet: a
// global variable captures the global context item; the match='.' template
// reports whether the global context item and the matched node are document
// nodes, plus the name of the global context item.
const globalContextXSL = `<xsl:stylesheet version='3.0' xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>
<xsl:variable name='v' select="."/>
<xsl:template match='.'><out root-is-doc="{$v instance of document-node()}" this-is-doc="{. instance of document-node()}"><xsl:value-of select='name($v)'/></out></xsl:template>
</xsl:stylesheet>`

// TestFnTransformGlobalContextItem covers fn-transform-82b/82c/82d: the initial
// match selection is the source-node itself (element vs document), while the
// global context item defaults to the root of the source-node unless an explicit
// global-context-item option overrides it.
func TestFnTransformGlobalContextItem(t *testing.T) {
	vars := xslXMLVars(globalContextXSL, "")

	testcases := []struct {
		name string
		expr string
	}{
		{
			// 82b: source-node is an element, no global-context-item. The global
			// context item defaults to the document root (root-is-doc=true) while
			// the template matches the element (this-is-doc=false, name="").
			name: "82b-element-source-default-gci-is-document",
			expr: `let $in := parse-xml("<dummy/>"),
			$r := transform(map{"source-node":$in/*,"stylesheet-text":$xsl,"xslt-version":3.0})?output
			return $r/out/@root-is-doc="true" and $r/out/@this-is-doc="false" and $r/out=""`,
		},
		{
			// 82c: source-node is a document, global-context-item is the element.
			// The global context item is the element (root-is-doc=false,
			// name="dummy"); the template matches the document (this-is-doc=true).
			name: "82c-document-source-element-gci",
			expr: `let $in := parse-xml("<dummy/>"),
			$r := transform(map{"source-node":$in,"global-context-item":$in/*,"stylesheet-text":$xsl,"xslt-version":3.0})?output
			return $r/out/@root-is-doc="false" and $r/out/@this-is-doc="true" and $r/out="dummy"`,
		},
		{
			// 82d: source-node is an element, global-context-item is the document.
			// The global context item is the document (root-is-doc=true, name="");
			// the template matches the element (this-is-doc=false).
			name: "82d-element-source-document-gci",
			expr: `let $in := parse-xml("<dummy/>"),
			$r := transform(map{"source-node":$in/*,"global-context-item":$in,"stylesheet-text":$xsl,"xslt-version":3.0})?output
			return $r/out/@root-is-doc="true" and $r/out/@this-is-doc="false" and $r/out=""`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, wantTrue, transformBool(t, tc.expr, vars))
		})
	}
}

// TestFnTransformNestedResultDocKey verifies that a nested xsl:result-document's
// key is its href resolved against the ENCLOSING result document's dynamic
// output URI, not the top-level base output URI.
func TestFnTransformNestedResultDocKey(t *testing.T) {
	xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'>
<xsl:result-document href='outer/index.html'><p>outer</p>
<xsl:result-document href='inner.html'><p>inner</p></xsl:result-document>
</xsl:result-document>
</xsl:template>
</xsl:stylesheet>`
	// The inner href resolves against the outer document's URI
	// (http://example.com/base/outer/index.html), yielding
	// http://example.com/base/outer/inner.html — NOT http://example.com/base/inner.html.
	expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml('<doc/>'),"base-output-uri":"http://example.com/base/main.xml"})
	return map:contains($r,"http://example.com/base/outer/index.html")
	   and map:contains($r,"http://example.com/base/outer/inner.html")
	   and not(map:contains($r,"http://example.com/base/inner.html"))`
	require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
}

// TestFnTransformSerializedTextPreservesTrailingNewline guards that method="text"
// serialized delivery keeps a legitimate trailing newline (only the xml-family
// serializer's document-terminating newline artifact is trimmed).
func TestFnTransformSerializedTextPreservesTrailingNewline(t *testing.T) {
	xsl := "<xsl:stylesheet version=\"3.0\" xmlns:xsl=\"http://www.w3.org/1999/XSL/Transform\">" +
		"<xsl:output method=\"text\"/>" +
		"<xsl:template name='main'>a\nb\n</xsl:template></xsl:stylesheet>"
	expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"delivery-format":"serialized"})?output
	return $r = concat("a", codepoints-to-string(10), "b", codepoints-to-string(10))`
	require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
}

// TestFnTransformGlobalContextItemTypeCheck verifies that an explicit
// fn:transform global-context-item (an item(), here an atomic value) is what
// gets type-checked against xsl:global-context-item/@as — not the source
// document. An integer matches as="xs:integer"; a string is an XTTE0590 error.
func TestFnTransformGlobalContextItemTypeCheck(t *testing.T) {
	xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:global-context-item as="xs:integer"/>
<xsl:template name="main"><out>ok</out></xsl:template>
</xsl:stylesheet>`
	vars := xslXMLVars(xsl, "")

	// An integer global-context-item matches as="xs:integer".
	okExpr := `let $r := transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"global-context-item":42})?output
	return $r/out = "ok"`
	require.Equal(t, wantTrue, transformBool(t, okExpr, vars))

	// A string global-context-item does not match as="xs:integer" → XTTE0590.
	badExpr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"global-context-item":"hello"})?output`
	_, err := evalTransform(t, badExpr, nil, vars, transformFns())
	require.Error(t, err)
	require.Contains(t, err.Error(), "XTTE0590")
}

// TestFnTransformSerializedNoTrailingNewline mirrors fn-transform-err-8: a
// serialized principal result must not carry a spurious trailing newline, so an
// ends-with test against the closing tag succeeds.
func TestFnTransformSerializedNoTrailingNewline(t *testing.T) {
	xsl := `<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:template name='main'><x>done</x></xsl:template>
</xsl:stylesheet>`
	expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"base-output-uri":"fn/transform/output.xml","delivery-format":"serialized"})?*
	return ends-with($r, "</x>")`
	require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
}

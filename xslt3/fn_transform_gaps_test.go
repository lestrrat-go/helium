package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestTransformPostProcess drives the fn:transform post-process callback for all
// three delivery formats, mirroring QT3 cases fn-transform-79 (document),
// fn-transform-80 (serialized), and fn-transform-81 (raw). The callback receives
// the result value and its return replaces the delivered output.
func TestTransformPostProcess(t *testing.T) {
	sourceDoc := helium.NewDefaultDocument()

	// fn-transform-79: delivery-format document. post-process navigates the
	// result document node and returns <b>89</b>, deep-equal to parse-xml('<b>89</b>')/*.
	t.Run("Document", func(t *testing.T) {
		expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='1.0'><xsl:template match='/'><out><xsl:copy-of select='.'/></out></xsl:template></xsl:stylesheet>" return
			let $expected := parse-xml('<b>89</b>')/* return
			let $trans-result := transform(map{"stylesheet-text":$xsl,
				"delivery-format":"document",
				"source-node": parse-xml('<a><b>89</b></a>'),
				"post-process": function($uri, $doc) { $doc/out/a/b }
				}) return
			deep-equal($trans-result("output"), $expected)`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})

	// fn-transform-80: delivery-format serialized. post-process truncates the
	// serialized string.
	t.Run("Serialized", func(t *testing.T) {
		expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='1.0'><xsl:template match='/'><out><xsl:copy-of select='.'/></out></xsl:template></xsl:stylesheet>" return
			let $trans-result := transform(map{"stylesheet-text":$xsl,
				"delivery-format":"serialized",
				"serialization-params": map { "method":"xml", "omit-xml-declaration":true(), "indent":false() },
				"source-node": parse-xml('<a><b>89</b></a>'),
				"post-process": function($uri, $out) { concat(substring($out, 1, 12), '...') }
				}) return
			deep-equal($trans-result("output"), "<out><a><b>8...")`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})

	// fn-transform-81: delivery-format raw. post-process arithmetic on the raw
	// atomic value (42 + 3 = 45).
	t.Run("Raw", func(t *testing.T) {
		expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='1.0'><xsl:template match='/'><xsl:sequence select='42'/></xsl:template></xsl:stylesheet>" return
			let $trans-result := transform(map{"stylesheet-text":$xsl,
				"delivery-format":"raw",
				"source-node": parse-xml('<a><b>89</b></a>'),
				"post-process": function($uri, $out) { $out + 3 }
				}) return
			deep-equal($trans-result("output"), 45)`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})
}

// TestTransformStylesheetNodeElement drives fn:transform with a stylesheet-node
// that is a bare (simplified / literal-result) element rather than a document
// node, mirroring QT3 cases fn-transform-7d and fn-transform-7e. The element is
// used as the stylesheet root even when it is not its owner document's document
// element.
func TestTransformStylesheetNodeElement(t *testing.T) {
	sourceDoc := helium.NewDefaultDocument()

	// fn-transform-7d: the simplified stylesheet <out> is the SECOND child of a
	// document fragment (the first is <noise/>); the source is a separate document.
	t.Run("FragmentElement", func(t *testing.T) {
		expr := `let $xsl := "<noise/>
			<out xmlns:xsl='http://www.w3.org/1999/XSL/Transform' xsl:version='2.0'>
			 <xsl:value-of select='.' />
			</out>" return
			transform(map{"stylesheet-node":parse-xml-fragment($xsl)/out, "source-node":parse-xml("<doc>this</doc>") })?output//out = 'this'`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})

	// fn-transform-7e: stylesheet and source are nodes in the SAME fragment
	// document; the stylesheet-node is the <out> element sibling of <doc>.
	t.Run("SameDocumentElement", func(t *testing.T) {
		expr := `let $src := parse-xml-fragment("<doc>this</doc>
			<out xmlns:xsl='http://www.w3.org/1999/XSL/Transform' xsl:version='2.0'>
			 <xsl:value-of select='/doc' />
			</out>") return
			transform(map{"stylesheet-node":$src/out, "source-node":$src })?output//out = 'this'`
		out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
		require.NoError(t, err)
		require.Equal(t, "true", out)
	})
}

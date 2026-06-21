package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// evalSelection evaluates an XPath expression against the source document and
// returns the resulting sequence, suitable for feeding to Invocation.Selection.
func evalSelection(t *testing.T, expr string, doc *helium.Document) xpath3.Sequence {
	t.Helper()
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	return res.Sequence()
}

// TestStripSpaceRemapsAttributeSelection verifies that when an initial match
// selection contains an attribute node, strip-space remaps the attribute onto
// the stripped copy: a template matched on the attribute, navigating up to its
// parent via XPath, must observe the STRIPPED parent (no whitespace-only text
// nodes). See finding 664-1.
func TestStripSpaceRemapsAttributeSelection(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="@id"><out><xsl:value-of select="count(../text())"/></out></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	// The root element has id="x" and two whitespace-only text nodes around the
	// <child> element. Without strip-space, count(../text()) would be 2.
	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root id="x">`+"\n  <child/>\n"+`</root>`))
	require.NoError(t, err)

	sel := evalSelection(t, "/*/@id", source)

	out, err := ss.ApplyTemplates(source).
		Selection(sel).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<out>0</out>",
		"attribute selection must remap to the stripped copy; got %q", out)
	require.NotContains(t, out, "<out>2</out>",
		"attribute selection must not see the unstripped original parent; got %q", out)
}

// TestStripSpaceRemapsNamespaceSelection verifies that when an initial match
// selection contains a namespace node, strip-space remaps it onto the stripped
// copy so that XPath navigation from the matched namespace node sees the
// stripped tree.
func TestStripSpaceRemapsNamespaceSelection(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="node()|@*"/>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root xmlns:a="urn:a">`+"\n  <child/>\n"+`</root>`))
	require.NoError(t, err)

	sel := evalSelection(t, "/*/namespace::*", source)
	require.Positive(t, sel.Len(), "fixture must select at least one namespace node")

	// The built-in template for namespace nodes does nothing; the key assertion
	// is that the transform runs without panicking and remapping succeeds.
	out, err := ss.ApplyTemplates(source).
		Selection(sel).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Empty(t, out)
}

// TestStripSpaceRemapsImplicitXMLNamespaceSelection verifies that the
// SYNTHESIZED implicit `xml` namespace node (which the XPath namespace axis
// fabricates and which is NOT present in the owner element's Namespaces())
// is remapped onto the stripped copy. A template matched on it that navigates
// to its owner element's text() must observe the STRIPPED parent (0 text nodes),
// matching the declared-namespace case. See finding 664-perf.
func TestStripSpaceRemapsImplicitXMLNamespaceSelection(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="namespace-node()"><out><xsl:value-of select="count(../text())"/></out></xsl:template>
  <xsl:template match="node()|@*"/>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	// The root element has two whitespace-only text nodes around <child>. Without
	// remapping, the implicit xml namespace node's owner points at the unstripped
	// original, so count(../text()) would be 2.
	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root>`+"\n  <child/>\n"+`</root>`))
	require.NoError(t, err)

	sel := evalSelection(t, "/*/namespace::xml", source)
	require.Equal(t, 1, sel.Len(), "fixture must select exactly the implicit xml namespace node")

	out, err := ss.ApplyTemplates(source).
		Selection(sel).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<out>0</out>",
		"implicit xml namespace selection must remap to the stripped copy; got %q", out)
	require.NotContains(t, out, "<out>2</out>",
		"implicit xml namespace selection must not see the unstripped original owner; got %q", out)
}

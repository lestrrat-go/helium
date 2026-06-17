package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestAttributeUndeclaredPrefixSequenceMode verifies that xsl:attribute with a
// computed name using an undeclared prefix raises XTDE0860 even when the
// attribute is constructed in sequence mode (xsl:variable/xsl:param with an
// "as" type), rather than being captured silently as a no-namespace attribute.
func TestAttributeUndeclaredPrefixSequenceMode(t *testing.T) {
	ctx := t.Context()

	// The variable has an "as" type, so xsl:attribute is constructed in
	// sequence mode. The computed name "p:a" uses prefix "p" which is not
	// declared anywhere in scope, so XTDE0860 must be raised.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="v" as="attribute()*">
      <xsl:attribute name="{'p:a'}" select="'x'"/>
    </xsl:variable>
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	_, err = ss.Transform(src).Serialize(ctx)
	require.Error(t, err, "undeclared prefix in computed attribute name must raise an error")
	require.True(t, strings.Contains(err.Error(), "XTDE0860"),
		"expected XTDE0860, got: %v", err)
}

// TestAttributeUndeclaredPrefixItemCapture verifies that xsl:attribute with a
// computed name using an undeclared prefix raises XTDE0860 when the attribute is
// captured as a standalone item (here via an item-serialization output method
// that captures the result rather than building a tree).
func TestAttributeUndeclaredPrefixItemCapture(t *testing.T) {
	ctx := t.Context()

	// method="adaptive" is an item-serialization method, so an xsl:attribute
	// constructed directly under the document node is captured as a pending
	// item. The computed name "p:a" uses an undeclared prefix "p", so XTDE0860
	// must be raised.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:attribute name="{'p:a'}" select="'x'"/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	_, err = ss.Transform(src).Serialize(ctx)
	require.Error(t, err, "undeclared prefix in computed attribute name must raise an error")
	require.True(t, strings.Contains(err.Error(), "XTDE0860"),
		"expected XTDE0860, got: %v", err)
}

// TestAttributeInvalidQNameSequenceMode verifies that xsl:attribute with a
// computed name that is not a lexically valid QName raises XTDE0850 in sequence
// mode (xsl:variable/xsl:param with an "as" type), rather than silently
// producing an attribute with an invalid name.
func TestAttributeInvalidQNameSequenceMode(t *testing.T) {
	ctx := t.Context()

	// "1bad" is not a valid NCName/QName (NCNames cannot start with a digit),
	// so XTDE0850 must be raised even though the attribute is constructed in
	// sequence mode.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="v" as="attribute()*">
      <xsl:attribute name="{'1bad'}" select="'x'"/>
    </xsl:variable>
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	_, err = ss.Transform(src).Serialize(ctx)
	require.Error(t, err, "invalid QName in computed attribute name must raise an error")
	require.True(t, strings.Contains(err.Error(), "XTDE0850"),
		"expected XTDE0850, got: %v", err)
}

// TestAttributeInvalidQNameItemCapture verifies that xsl:attribute with a
// computed name that is not a lexically valid QName raises XTDE0850 when the
// attribute is captured as a standalone item via an item-serialization output
// method.
func TestAttributeInvalidQNameItemCapture(t *testing.T) {
	ctx := t.Context()

	// As above, "1bad" is not a valid QName. With method="adaptive" the
	// attribute is captured as a pending item, and XTDE0850 must still be
	// raised.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:attribute name="{'1bad'}" select="'x'"/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	_, err = ss.Transform(src).Serialize(ctx)
	require.Error(t, err, "invalid QName in computed attribute name must raise an error")
	require.True(t, strings.Contains(err.Error(), "XTDE0850"),
		"expected XTDE0850, got: %v", err)
}

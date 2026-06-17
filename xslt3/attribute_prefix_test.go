package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
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

// TestAttributeExplicitNamespaceSequenceMode verifies that an xsl:attribute with
// a computed name using an undeclared prefix BUT an explicit namespace= attribute
// is assigned that namespace (not no-namespace) when constructed in sequence mode.
func TestAttributeExplicitNamespaceSequenceMode(t *testing.T) {
	ctx := t.Context()

	// The prefix "p" is undeclared, but namespace="urn:p" is supplied, so the
	// attribute must be in urn:p, not in no-namespace. We capture it in a
	// sequence-typed variable and emit its namespace URI as text.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:variable name="v" as="attribute()*">
      <xsl:attribute name="{'p:a'}" namespace="urn:p" select="'x'"/>
    </xsl:variable>
    <xsl:value-of select="namespace-uri-from-QName(node-name($v[1]))"/>
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
	require.Equal(t, "urn:p", strings.TrimSpace(out),
		"computed attribute with explicit namespace= must be in that namespace in sequence mode")
}

// primaryItemsCapture is a PrimaryItemsHandler that records the items captured
// from the primary output so a test can inspect captured attribute nodes.
type primaryItemsCapture struct {
	seq xpath3.Sequence
}

func (p *primaryItemsCapture) HandlePrimaryItems(seq xpath3.Sequence) error {
	p.seq = seq
	return nil
}

// TestAttributeExplicitNamespaceItemCapture verifies that an xsl:attribute with a
// computed name using an undeclared prefix BUT an explicit namespace= attribute
// is assigned that namespace (not no-namespace) when captured as a standalone
// item via an item-serialization output method (the item-capture path). The
// captured attribute node's namespace URI is inspected via a PrimaryItemsHandler.
func TestAttributeExplicitNamespaceItemCapture(t *testing.T) {
	ctx := t.Context()

	// method="adaptive" is an item-serialization method, so the standalone
	// xsl:attribute at the top level is captured as a pending item rather than
	// attached to an element. The undeclared prefix "p" is overridden by
	// namespace="urn:p", so the captured attribute node must be in urn:p.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:attribute name="{'p:a'}" namespace="urn:p" select="'x'"/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	capture := &primaryItemsCapture{}
	_, err = ss.Transform(src).PrimaryItemsHandler(capture).Do(ctx)
	require.NoError(t, err)

	require.NotNil(t, capture.seq, "expected primary items to be captured")
	require.Equal(t, 1, capture.seq.Len(), "expected a single captured attribute item")
	ni, ok := capture.seq.Get(0).(xpath3.NodeItem)
	require.True(t, ok, "captured item must be a node item")
	attr, ok := helium.AsNode[*helium.Attribute](ni.Node)
	require.True(t, ok, "captured node must be an attribute")
	require.Equal(t, "a", attr.LocalName())
	require.Equal(t, "urn:p", attr.URI(),
		"computed attribute with explicit namespace= must be in that namespace in item-capture mode")
}

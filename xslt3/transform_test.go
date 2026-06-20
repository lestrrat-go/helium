package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestCallTemplateCoercesParamsToDeclaredTypes(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:call-template name="show">
        <xsl:with-param name="a" select="xs:untypedAtomic('FOO')"/>
        <xsl:with-param name="c" select="xs:untypedAtomic('50')"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="show">
    <xsl:param name="a" as="xs:string"/>
    <xsl:param name="c" as="xs:double"/>
    <q a="{$a instance of xs:string}" c="{$c instance of xs:double}"/>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, `a="true"`)
	require.Contains(t, result, `c="true"`)
}

// TestGlobalParamStaticBaseURI verifies that static-base-uri() inside a
// global param's select resolves against the declaration-site xml:base,
// not the stylesheet's base URI.
func TestGlobalParamStaticBaseURI(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="base" xml:base="http://example.com/override/"
      select="static-base-uri()"/>
  <xsl:template match="/">
    <out><xsl:value-of select="$base"/></out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "http://example.com/override/")
}

// TestCommentBodyNoStraySpace verifies that xsl:comment body construction
// does not produce a stray leading space when an empty TVT precedes text.
func TestCommentBodyNoStraySpace(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="empty" select="''"/>
  <xsl:template match="/">
    <out>
      <xsl:comment>
        <xsl:value-of select="$empty"/>
        <xsl:text>hello</xsl:text>
      </xsl:comment>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	// The comment content should be "hello" with no leading space.
	require.Contains(t, out, "<!--hello-->")
}

// TestDocVariableInterleavesSequence verifies that a document-node variable
// body preserves document order between literal result elements and
// xsl:sequence outputs (constructed nodes and atomics interleaved).
func TestDocVariableInterleavesSequence(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "atomic between elements",
			body: `<a/><xsl:sequence select="'b'"/><c/>`,
			want: "<out><a/>b<c/></out>",
		},
		{
			name: "node from sequence between elements",
			body: `<a/><xsl:sequence select="//src"/><c/>`,
			want: "<out><a/><src/><c/></out>",
		},
		{
			// xsl:sequence select="/" yields a document node; its children
			// (the source root element) must be spliced in document order,
			// not the document node itself.
			name: "document node between elements",
			body: `<a/><xsl:sequence select="/"/><c/>`,
			want: "<out><a/><doc><src/></doc><c/></out>",
		},
		{
			name: "multiple atomics interleaved",
			body: `<xsl:sequence select="1"/><a/><xsl:sequence select="2"/><b/><xsl:sequence select="3"/>`,
			want: "<out>1<a/>2<b/>3</out>",
		},
		{
			name: "trailing element after sequence",
			body: `<xsl:sequence select="('x','y')"/><z/>`,
			want: "<out>x y<z/></out>",
		},
		{
			// xsl:try select also captures into the document; it must keep
			// document order with surrounding literal result elements, not be
			// appended after them.
			name: "try select between elements",
			body: `<a/><xsl:try select="'b'"><xsl:catch select="'x'"/></xsl:try><c/>`,
			want: "<out><a/>b<c/></out>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="v">` + tc.body + `</xsl:variable>
    <out><xsl:copy-of select="$v"/></out>
  </xsl:template>
</xsl:stylesheet>`

			doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
			require.NoError(t, err)
			ss, err := xslt3.CompileStylesheet(ctx, doc)
			require.NoError(t, err)
			src, _ := helium.NewParser().Parse(ctx, []byte(`<doc><src/></doc>`))
			out, err := ss.Transform(src).Serialize(ctx)
			require.NoError(t, err)
			require.Contains(t, out, tc.want)
		})
	}
}

// TestDocVariableDocumentNodeSequenceSplicesChildren verifies the structural
// shape of a document variable built with xsl:sequence select="/" interleaved
// with literal elements. The document node must contribute its children (the
// source root element) spliced in document order, so $v/node() sees three
// nodes (a, doc, c) with the source root in the middle — not a nested document
// node that would collapse $v/node() to two (a, c).
func TestDocVariableDocumentNodeSequenceSplicesChildren(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="v"><a/><xsl:sequence select="/"/><c/></xsl:variable>
    <out count="{count($v/node())}" mid="{local-name($v/node()[2])}"/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<doc><src/></doc>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, `count="3"`)
	require.Contains(t, out, `mid="doc"`)
}

// TestDocVariableMergesAdjacentText verifies that text produced by xsl:sequence
// adjacent to text from xsl:text/xsl:value-of is merged into a single text node
// in the constructed document tree (XSLT result-tree construction merges
// adjacent text nodes), so node-level XPath sees one text node, not two.
func TestDocVariableMergesAdjacentText(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name string
		body string
	}{
		{name: "sequence then text", body: `<xsl:sequence select="'a'"/><xsl:text>b</xsl:text>`},
		{name: "text then sequence", body: `<xsl:text>a</xsl:text><xsl:sequence select="'b'"/>`},
		{name: "text sequence text", body: `<xsl:text>a</xsl:text><xsl:sequence select="'b'"/><xsl:text>c</xsl:text>`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="v">` + tc.body + `</xsl:variable>
    <out count="{count($v/text())}"><xsl:value-of select="string($v)"/></out>
  </xsl:template>
</xsl:stylesheet>`

			doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
			require.NoError(t, err)
			ss, err := xslt3.CompileStylesheet(ctx, doc)
			require.NoError(t, err)
			src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
			out, err := ss.Transform(src).Serialize(ctx)
			require.NoError(t, err)
			require.Contains(t, out, `count="1"`)
		})
	}
}

// TestDocVariableTypedTemplateResultOrder verifies that the result of a typed
// (as="...") template invoked via xsl:call-template inside a document-variable
// body keeps document order with surrounding literal result elements, rather
// than being appended after them.
func TestDocVariableTypedTemplateResultOrder(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <xsl:variable name="v"><a/><xsl:call-template name="emit"/><c/></xsl:variable>
    <out><xsl:copy-of select="$v"/></out>
  </xsl:template>
  <xsl:template name="emit" as="xs:string"><xsl:sequence select="'b'"/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Regexp(t, `<a[^>]*/>b<c[^>]*/>`, out)
}

// TestDocVariableNestedSequenceNoPlaceholderLeak verifies that an xsl:sequence
// nested inside an xsl:copy in a document-variable body writes into the copied
// element directly (not via a document-level placeholder). The copied element
// becomes the temp tree's document element, so the placeholder capture path
// must not fire there — otherwise the unresolved placeholder PI leaks into
// output.
func TestDocVariableNestedSequenceNoPlaceholderLeak(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="v">
      <xsl:for-each select="/doc/src">
        <xsl:copy><xsl:sequence select="'b'"/></xsl:copy>
      </xsl:for-each>
    </xsl:variable>
    <out><xsl:copy-of select="$v"/></out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<doc><src/></doc>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "<out><src>b</src></out>")
	require.NotContains(t, out, "helium-xsl-sequence-placeholder")
}

// TestPIBodyNoStraySpace verifies that xsl:processing-instruction body
// does not produce a stray leading space when an empty TVT precedes text.
func TestPIBodyNoStraySpace(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="empty" select="''"/>
  <xsl:template match="/">
    <out>
      <xsl:processing-instruction name="target">
        <xsl:value-of select="$empty"/>
        <xsl:text>data</xsl:text>
      </xsl:processing-instruction>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	// The PI content should be "data" with no leading space.
	require.Contains(t, out, "<?target data?>")
}

func TestAnnotateAttrRegistersIDSubtype(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:simpleType name="myID">
        <xs:restriction base="xs:ID"/>
      </xs:simpleType>
      <xs:complexType name="rootType">
        <xs:attribute name="id" type="myID"/>
      </xs:complexType>
      <xs:element name="root" type="rootType"/>
    </xs:schema>
  </xsl:import-schema>

  <xsl:template match="/">
    <result>
      <found><xsl:value-of select="boolean(id('alpha'))"/></found>
      <name><xsl:value-of select="id('alpha')/name()"/></name>
    </result>
  </xsl:template>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<root id="alpha"/>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	require.Contains(t, result, "<found>true</found>")
	require.Contains(t, result, "<name>root</name>")
}

// TestIterateAtomicClearsNodeContext verifies that xsl:iterate over a
// sequence of atomic items sets the context item without leaving a stale
// node context. xsl:copy inside the body must copy the current atomic item
// as text, not the previous source node.
func TestIterateAtomicClearsNodeContext(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:iterate select="'x'"><xsl:copy/></xsl:iterate></out>
  </xsl:template>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	require.Contains(t, result, "<out>x</out>")
	require.NotContains(t, result, "<doc/>")
}

// TestGlobalContextItemNamespaceAwareType verifies that an xsl:global-context-item
// declared as="document-node(element(p:root))" is validated namespace-aware: a
// document whose root is in the wrong namespace is rejected (XTTE0590) and one
// with the correctly-namespaced root is accepted.
func TestGlobalContextItemNamespaceAwareType(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:p="urn:right">
  <xsl:global-context-item as="document-node(element(p:root))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="name(/*)"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	wrong, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns="urn:wrong"/>`))
	require.NoError(t, err)
	_, err = xslt3.TransformString(t.Context(), wrong, ss)
	require.Error(t, err, "wrong-namespace root must be rejected")

	right, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns="urn:right"/>`))
	require.NoError(t, err)
	result, err := xslt3.TransformString(t.Context(), right, ss)
	require.NoError(t, err, "correctly-namespaced root must be accepted")
	require.Contains(t, result, "root")
}

// TestGlobalContextItemDeclarationLocalNamespace verifies that the @as type on
// xsl:global-context-item resolves prefixes against the declaration element's
// own namespace context, not the runtime stylesheet-wide context. Here the p:
// prefix is declared on the xsl:global-context-item element itself.
func TestGlobalContextItemDeclarationLocalNamespace(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:global-context-item xmlns:p="urn:right" as="document-node(element(p:root))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="name(/*)"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	right, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns="urn:right"/>`))
	require.NoError(t, err)
	result, err := xslt3.TransformString(t.Context(), right, ss)
	require.NoError(t, err, "default-namespaced root must match declaration-local prefix")
	require.Contains(t, result, "root")

	right2, err := helium.NewParser().Parse(t.Context(), []byte(`<p:root xmlns:p="urn:right"/>`))
	require.NoError(t, err)
	_, err = xslt3.TransformString(t.Context(), right2, ss)
	require.NoError(t, err, "explicitly-prefixed {urn:right}root must also match")

	wrong, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns="urn:wrong"/>`))
	require.NoError(t, err)
	_, err = xslt3.TransformString(t.Context(), wrong, ss)
	require.Error(t, err, "wrong-namespace root must be rejected")
}

// TestGlobalContextItemXPathDefaultNamespace verifies that the
// xpath-default-namespace in scope at the xsl:global-context-item declaration
// is used to resolve the unprefixed element name in its @as type.
func TestGlobalContextItemXPathDefaultNamespace(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:global-context-item xpath-default-namespace="urn:right"
    as="document-node(element(root))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="local-name(/*)"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	right, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns="urn:right"/>`))
	require.NoError(t, err)
	result, err := xslt3.TransformString(t.Context(), right, ss)
	require.NoError(t, err, "root in xpath-default-namespace must be accepted")
	require.Contains(t, result, "root")

	wrong, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns="urn:wrong"/>`))
	require.NoError(t, err)
	_, err = xslt3.TransformString(t.Context(), wrong, ss)
	require.Error(t, err, "root in wrong namespace must be rejected")
}

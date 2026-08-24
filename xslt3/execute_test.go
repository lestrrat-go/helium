package xslt3_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
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

// TestApplyTemplatesMixedSelectionOrder verifies that xsl:apply-templates
// processes a mixed sequence of atomic values and nodes in sequence order,
// not by processing all nodes before all atomic values. Per XSLT 3.0, the
// selected sequence is processed in order.
func TestApplyTemplatesMixedSelectionOrder(t *testing.T) {
	ctx := t.Context()

	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="('a', /root/b, 'c')"/></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="b">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><b>B</b></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	// Sequence order is: 'a', /root/b, 'c'. The buggy implementation emits
	// all nodes first, producing [node:B][str:a][str:c].
	require.Contains(t, out, `[str:a][node:B][str:c]`)
}

// TestApplyTemplatesMixedSortPosition verifies that within an xsl:sort over a
// mixed atomic+node selection, position()/last() in the sort key reflect the
// full mixed sequence (size = number of selected items, position = 1-based
// index in the unsorted sequence), and never a stale size of 1.
func TestApplyTemplatesMixedSortPosition(t *testing.T) {
	ctx := t.Context()

	// Selection is ('x', /root/a, 'y'), three items. Sort key = position()
	// in descending order, so the processing order must be reversed:
	// 'y' (pos 3), /root/a (pos 2), 'x' (pos 1).
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="('x', /root/a, 'y')">
      <xsl:sort select="position()" data-type="number" order="descending"/>
    </xsl:apply-templates></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="a">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><a>A</a></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Contains(t, out, `[str:y][node:A][str:x]`)
}

// TestApplyTemplatesMixedSortLast verifies that last() in the sort key over a
// mixed atomic+node selection reports the full mixed sequence size.
func TestApplyTemplatesMixedSortLast(t *testing.T) {
	ctx := t.Context()

	// last() must be 3 for every item. Sort key = (last() - position()) so the
	// order is reversed: 'y' (3-3=0), /root/a (3-2=1), 'x' (3-1=2) ascending.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="('x', /root/a, 'y')">
      <xsl:sort select="last() - position()" data-type="number"/>
    </xsl:apply-templates></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="a">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><a>A</a></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Contains(t, out, `[str:y][node:A][str:x]`)
}

// TestApplyTemplatesMixedSortCurrent verifies that current() in the sort key
// resolves to the node being sorted for NODE items in a mixed selection, while
// non-node items still atomize via the context item.
func TestApplyTemplatesMixedSortCurrent(t *testing.T) {
	ctx := t.Context()

	// Selection mixes nodes and a string. Sort by current() string value so
	// node items sort by their own value via current(). Nodes a/b/c have
	// values "3","1","2"; string "0" sorts first. Expected ascending order:
	// '0', b(1), c(2), a(3).
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="(/root/a, '0', /root/b, /root/c)">
      <xsl:sort select="current()" data-type="number"/>
    </xsl:apply-templates></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="*">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><a>3</a><b>1</b><c>2</c></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Contains(t, out, `[str:0][node:1][node:2][node:3]`)
}

// TestForEachGroupStartingWithPositionalPattern verifies that a positional
// pattern in group-starting-with sees the per-item focus (position/size of the
// population sequence), and never the stale outer focus (ENG-005). The
// population is an atomic sequence, so the pattern predicate is evaluated with
// the item as context and reads ec.position/ec.size. With the bug, position()=3
// never matches (position stuck at the outer 1), producing a single group; the
// fix yields two groups split before the 3rd item.
func TestForEachGroupStartingWithPositionalPattern(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="(1,2,3,4,5)" group-starting-with=".[position()=3]">
        <group><xsl:value-of select="string-join(current-group()!string(.), ',')"/></group>
      </xsl:for-each-group>
    </out>
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

	require.Equal(t, 2, strings.Count(out, "<group>"),
		"positional pattern should split into two groups, got: %s", out)
	require.Contains(t, out, "<group>1,2</group>")
	require.Contains(t, out, "<group>3,4,5</group>")
}

// TestForEachGroupEndingWithPositionalPattern verifies the same per-item focus
// handling for group-ending-with (ENG-005). position()=3 ends a group at the
// 3rd item of the population.
func TestForEachGroupEndingWithPositionalPattern(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="(1,2,3,4,5)" group-ending-with=".[position()=3]">
        <group><xsl:value-of select="string-join(current-group()!string(.), ',')"/></group>
      </xsl:for-each-group>
    </out>
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

	require.Equal(t, 2, strings.Count(out, "<group>"),
		"positional pattern should split into two groups, got: %s", out)
	require.Contains(t, out, "<group>1,2,3</group>")
	require.Contains(t, out, "<group>4,5</group>")
}

// TestForEachGroupStartingWithNodePositionalPattern verifies that a positional
// pattern in group-starting-with sees the per-item focus of the population when
// the population is a sequence of element NODES (UNRES-7, #684 follow-up). The
// node path previously delegated straight to matchPattern, which re-established
// the node's document-order focus and ignored ec.position/ec.size, so
// position()=3 never matched and the items collapsed into a single group. The
// fix routes ".[pred]" alternatives through the population focus, splitting
// before the 3rd node.
func TestForEachGroupStartingWithNodePositionalPattern(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="root/item" group-starting-with=".[position()=3]">
        <group><xsl:value-of select="string-join(current-group()!string(@n), ',')"/></group>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(ctx, []byte(
		`<root><item n="1"/><item n="2"/><item n="3"/><item n="4"/><item n="5"/></root>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Equal(t, 2, strings.Count(out, "<group>"),
		"node positional pattern should split into two groups, got: %s", out)
	require.Contains(t, out, "<group>1,2</group>")
	require.Contains(t, out, "<group>3,4,5</group>")
}

// TestForEachGroupEndingWithNodePositionalPattern verifies the same per-item
// focus handling for group-ending-with over element NODES (UNRES-7). A group
// ends at the 3rd node of the population.
func TestForEachGroupEndingWithNodePositionalPattern(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="root/item" group-ending-with=".[position()=3]">
        <group><xsl:value-of select="string-join(current-group()!string(@n), ',')"/></group>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(ctx, []byte(
		`<root><item n="1"/><item n="2"/><item n="3"/><item n="4"/><item n="5"/></root>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Equal(t, 2, strings.Count(out, "<group>"),
		"node positional pattern should split into two groups, got: %s", out)
	require.Contains(t, out, "<group>1,2,3</group>")
	require.Contains(t, out, "<group>4,5</group>")
}

// TestForEachGroupStartingWithNumericLiteralPattern verifies the numeric-literal
// positional predicate (the atomic branch of matchContextItemPredicates) does an
// exact float compare, not a truncating int compare. position()=2.7 is never
// true, so ".[2.7]" must not start a group anywhere over (1,2,3,4,5), yielding a
// single group; ".[3]" and ".[3.0]" both match position 3 exactly and split.
func TestForEachGroupStartingWithNumericLiteralPattern(t *testing.T) {
	ctx := t.Context()
	tmpl := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="(1,2,3,4,5)" group-starting-with=".[%s]">
        <group><xsl:value-of select="string-join(current-group()!string(.), ',')"/></group>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	run := func(t *testing.T, pred string) string {
		t.Helper()
		xsltSrc := strings.Replace(tmpl, "%s", pred, 1)
		doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
		require.NoError(t, err)
		ss, err := xslt3.CompileStylesheet(ctx, doc)
		require.NoError(t, err)
		src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
		require.NoError(t, err)
		out, err := ss.Transform(src).Serialize(ctx)
		require.NoError(t, err)
		return out
	}

	t.Run("fractional literal never matches", func(t *testing.T) {
		out := run(t, "2.7")
		require.Equal(t, 1, strings.Count(out, "<group>"),
			"position()=2.7 is never true, expected a single group, got: %s", out)
		require.Contains(t, out, "<group>1,2,3,4,5</group>")
	})

	t.Run("integer literal matches position", func(t *testing.T) {
		out := run(t, "3")
		require.Equal(t, 2, strings.Count(out, "<group>"),
			".[3] should split at position 3, got: %s", out)
		require.Contains(t, out, "<group>1,2</group>")
		require.Contains(t, out, "<group>3,4,5</group>")
	})

	t.Run("integer-valued float matches position", func(t *testing.T) {
		out := run(t, "3.0")
		require.Equal(t, 2, strings.Count(out, "<group>"),
			".[3.0] should split at position 3, got: %s", out)
		require.Contains(t, out, "<group>1,2</group>")
		require.Contains(t, out, "<group>3,4,5</group>")
	})
}

// ENG-001: a template rule matching an ATOMIC item with a required param
// supplied via xsl:with-param must succeed (no XTDE0700) and the param value
// must be visible in the template body.
func TestApplyTemplatesAtomicRequiredParamSupplied(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="1">
        <xsl:with-param name="p" select="'supplied'"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match=".[. instance of xs:integer]">
    <xsl:param name="p" as="xs:string" required="yes"/>
    <got><xsl:value-of select="$p"/></got>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, "<got>supplied</got>")
}

// ENG-002: a caller-supplied empty sequence () for a param with
// as="xs:string" (cardinality exactly-one) must raise XTTE0590, not pass
// silently.
func TestCallTemplateEmptySequenceForExactlyOneParamFails(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:call-template name="show">
        <xsl:with-param name="p" select="()"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="show">
    <xsl:param name="p" as="xs:string"/>
    <q><xsl:value-of select="$p"/></q>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	_, err := xslt3.TransformString(t.Context(), source, ss)
	require.Error(t, err)
	require.Contains(t, err.Error(), "XTTE0590")
}

// Variables declared inside xsl:try (or its xsl:catch) must not leak into the
// surrounding scope. After the instruction completes, an outer variable of the
// same name must still resolve to its outer value.
func TestTryDoesNotLeakVariables(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			// Try body succeeds; inner $x must not shadow outer $x afterward.
			name: "success",
			body: `
      <xsl:try>
        <xsl:variable name="x" select="'inner'"/>
        <xsl:catch/>
      </xsl:try>`,
		},
		{
			// Try body fails; catch runs and declares $x, which must not leak.
			name: "catch",
			body: `
      <xsl:try>
        <xsl:sequence select="1 div xs:integer('not-a-number')"/>
        <xsl:catch>
          <xsl:variable name="x" select="'inner'"/>
        </xsl:catch>
      </xsl:try>`,
		},
		{
			// rollback-output="no" with a successful try body.
			name: "no-rollback-success",
			body: `
      <xsl:try rollback-output="no">
        <xsl:variable name="x" select="'inner'"/>
        <xsl:catch/>
      </xsl:try>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <xsl:variable name="x" select="'outer'"/>
    <out>`+tc.body+`<xsl:value-of select="$x"/></out>
  </xsl:template>
</xsl:stylesheet>`)

			result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Contains(t, result, ">outer<")
			require.NotContains(t, result, ">inner<")
		})
	}
}

// A tunnel parameter set by an xsl:call-template whose with-param evaluation
// later fails must not leak into templates invoked from the surrounding
// xsl:catch. The tunnel param is evaluated (mutating the active tunnel map)
// before a sibling with-param raises a dynamic error; the error is caught, and
// a second template called from xsl:catch must see the tunnel param as absent.
func TestCallTemplateTunnelParamDoesNotLeakAcrossCaughtError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:try>
        <xsl:call-template name="consume">
          <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
          <xsl:with-param name="bad" select="xs:integer('not-a-number')"/>
        </xsl:call-template>
        <xsl:catch>
          <xsl:call-template name="check"/>
        </xsl:catch>
      </xsl:try>
    </out>
  </xsl:template>

  <xsl:template name="consume">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="bad"/>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "NOLEAK")
	require.NotContains(t, result, "LEAKED")
}

// A tunnel parameter must not become observable to a template invoked from a
// LATER sibling with-param's BODY before control is actually transferred to the
// target template. Here the first with-param sets the tunnel param, and a later
// with-param body contains an xsl:try whose error is caught; the xsl:catch calls
// another template that reads the same tunnel param. Because the value is still
// being assembled (the call/next-match/apply-imports has not yet handed control
// to its target), the called template must see the tunnel param as ABSENT.
// This is the two-phase guarantee that xsl:apply-templates already provides.
func TestTunnelParamNotObservedByLaterWithParamBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{
			name: "call-template",
			src: `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:call-template name="consume">
        <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
        <xsl:with-param name="probe">
          <xsl:try>
            <xsl:sequence select="xs:integer('not-a-number')"/>
            <xsl:catch>
              <xsl:call-template name="check"/>
            </xsl:catch>
          </xsl:try>
        </xsl:with-param>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="consume">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="probe"/>
    <xsl:copy-of select="$probe"/>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`,
		},
		{
			name: "next-match",
			src: `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="root" priority="2">
    <out>
      <xsl:next-match>
        <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
        <xsl:with-param name="probe">
          <xsl:try>
            <xsl:sequence select="xs:integer('not-a-number')"/>
            <xsl:catch>
              <xsl:call-template name="check"/>
            </xsl:catch>
          </xsl:try>
        </xsl:with-param>
      </xsl:next-match>
    </out>
  </xsl:template>

  <xsl:template match="root" priority="1">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="probe"/>
    <xsl:copy-of select="$probe"/>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ss := compileStylesheetString(t, tc.src)
			result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Contains(t, result, "NOLEAK")
			require.NotContains(t, result, "LEAKED")
		})
	}
}

// Same two-phase guarantee for xsl:apply-imports: a tunnel with-param set by an
// earlier sibling must not be visible to a template invoked from a later
// with-param body whose inner error is caught, before control is transferred to
// the imported template.
func TestApplyImportsTunnelParamNotObservedByLaterWithParamBody(t *testing.T) {
	const importedURI = "mem:/imported-tunnel.xsl"

	imported := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root">
    <xsl:param name="tp" tunnel="yes"/>
    <xsl:param name="probe"/>
    <xsl:copy-of select="$probe"/>
  </xsl:template>
</xsl:stylesheet>`

	main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import href="` + importedURI + `"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="root">
    <out>
      <xsl:apply-imports>
        <xsl:with-param name="tp" select="'LEAKED'" tunnel="yes"/>
        <xsl:with-param name="probe">
          <xsl:try>
            <xsl:sequence select="xs:integer('not-a-number')"/>
            <xsl:catch>
              <xsl:call-template name="check"/>
            </xsl:catch>
          </xsl:try>
        </xsl:with-param>
      </xsl:apply-imports>
    </out>
  </xsl:template>

  <xsl:template name="check">
    <xsl:param name="tp" select="'NOLEAK'" tunnel="yes"/>
    <xsl:value-of select="$tp"/>
  </xsl:template>
</xsl:stylesheet>`

	resolver := &memResolver{files: map[string]string{importedURI: imported}}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI("mem:/main.xsl").
		URIResolver(resolver).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte("<root/>"))
	require.NoError(t, err)

	out, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, out, "NOLEAK")
	require.NotContains(t, out, "LEAKED")
}

// TestCopyOfAttributeValuesNotReparsed guards against the result-tree copy path
// re-parsing an already-resolved attribute value. xsl:copy-of of an element (and
// xsl:copy) duplicates the source element into the result tree via a deep copy;
// the attribute value returned by the parser is already entity-resolved, so it
// must be stored LITERALLY (and re-escaped by the serializer), and never fed
// back through a value-parsing setter (SetParsedAttribute), which would choke on
// a bare '&' (an "entity was unterminated" error) or silently double-resolve
// '&amp;amp;'.
func TestCopyOfAttributeValuesNotReparsed(t *testing.T) {
	t.Parallel()

	// srcAttr is the lexical attribute value as authored in the source XML;
	// wantAttr is its expected serialization after an entity-resolved round trip.
	cases := []struct {
		name     string
		srcAttr  string
		wantAttr string
	}{
		{"ampersand", "x&amp;y", "x&amp;y"},
		{"less-than", "a&lt;b", "a&lt;b"},
		{"greater-than", "a&gt;b", "a&gt;b"},
		{"quote", "&quot;q&quot;", "&quot;q&quot;"},
		{"numeric-ref", "&#65;&#66;", "AB"},
		{"double-escaped", "&amp;amp;", "&amp;amp;"},
		{"mixed", "p?a=1&amp;b=2&lt;3", "p?a=1&amp;b=2&lt;3"},
	}

	copyOfSheet := compileSheet(t, `<out><xsl:copy-of select="."/></out>`)
	copySheet := compileSheet(t, `<xsl:copy select="."><xsl:copy-of select="@*"/></xsl:copy>`)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<e a="`+tc.srcAttr+`"/>`))
			require.NoError(t, err)

			wantElem := `<e a="` + tc.wantAttr + `"/>`

			// xsl:copy-of of the element: the previously-broken path.
			out, err := xslt3.TransformString(t.Context(), src, copyOfSheet)
			require.NoError(t, err, "xsl:copy-of must not re-parse the resolved value")
			require.Equal(t, "<out>"+wantElem+"</out>", out)

			// xsl:copy of the element plus xsl:copy-of of its attributes.
			out2, err := xslt3.TransformString(t.Context(), src, copySheet)
			require.NoError(t, err, "xsl:copy must not re-parse the resolved value")
			require.Equal(t, wantElem, out2)
		})
	}
}

// TestCopyOfNamespacedAttributeNotReparsed exercises the namespaced-attribute
// branch of the deep-copy attribute loop (SetAttributeNS).
func TestCopyOfNamespacedAttributeNotReparsed(t *testing.T) {
	t.Parallel()

	src, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<e xmlns:p="urn:p" p:a="x&amp;y&lt;z"/>`))
	require.NoError(t, err)

	ss := compileSheet(t, `<out><xsl:copy-of select="."/></out>`)
	out, err := xslt3.TransformString(t.Context(), src, ss)
	require.NoError(t, err)
	require.Equal(t, `<out><e xmlns:p="urn:p" p:a="x&amp;y&lt;z"/></out>`, out)
}

// compileSheet compiles a minimal stylesheet whose single template body (matched
// on the /e source element) is the provided sequence constructor.
func compileSheet(t *testing.T, body string) *xslt3.Stylesheet {
	t.Helper()
	sheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output method="xml" omit-xml-declaration="yes"/>
<xsl:template match="/e">` + body + `</xsl:template>
</xsl:stylesheet>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(sheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	return ss
}

// TestAttributeUndeclaredPrefixSequenceMode verifies that xsl:attribute with a
// computed name using an undeclared prefix raises XTDE0860 even when the
// attribute is constructed in sequence mode (xsl:variable/xsl:param with an
// "as" type), and is never captured silently as a no-namespace attribute.
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
// that captures the result and builds no tree).
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
// mode (xsl:variable/xsl:param with an "as" type), producing no
// attribute with an invalid name.
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
	// xsl:attribute at the top level is captured as a pending item, and never
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

// analyzeStringStylesheet builds an xsl:analyze-string stylesheet using the
// given regex; the matching and non-matching substrings are emitted verbatim
// so output is easy to assert.
func analyzeStringStylesheet(regex string) string {
	return `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:output method="xml" omit-xml-declaration="yes"/>` +
		`<xsl:template match="/"><out>` +
		`<xsl:analyze-string select="string(.)" regex="` + regex + `">` +
		`<xsl:matching-substring><m><xsl:value-of select="."/></m></xsl:matching-substring>` +
		`<xsl:non-matching-substring><n><xsl:value-of select="."/></n></xsl:non-matching-substring>` +
		`</xsl:analyze-string>` +
		`</out></xsl:template>` +
		`</xsl:stylesheet>`
}

// A normal xsl:analyze-string still produces correct alternating
// matching / non-matching output. This pins byte-identical behavior across the
// incremental-processing change.
func TestAnalyzeStringNormalOutput(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(analyzeStringStylesheet("[0-9]")))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>a1b2c3</doc>`))
	require.NoError(t, err)

	result, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result,
		"<out><n>a</n><m>1</m><n>b</n><m>2</m><n>c</n><m>3</m></out>")
}

// An xsl:analyze-string with an empty-matching regex over a large input matches
// at every character boundary, amplifying a bounded input string into an
// unbounded number of match/segment allocations. The work must be bounded
// against the execution resource budget (MaxResourceBytes) and fail with
// ErrResourceTooLarge, exhausting no memory.
func TestAnalyzeStringEmptyMatchIsCapped(t *testing.T) {
	t.Parallel()

	// regex "x*" matches a zero-length string at every position of an all-'a'
	// input, so an L-char input yields L+1 matches.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(analyzeStringStylesheet("x*")))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc>`+strings.Repeat("a", 5000)+`</doc>`))
	require.NoError(t, err)

	// Cap well below the resulting match count; the breach must surface
	// ErrResourceTooLarge through the dynamic-error wrapping.
	_, err = ss.Transform(source).
		MaxResourceBytes(1000).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"empty-matching xsl:analyze-string over a large input must honor the resource cap")
	require.ErrorIs(t, err, xslt3.ErrDynamicError,
		"the analyze-string cap breach is a runtime (dynamic) error")
}

// A multi-line "^" (flags="m") is a leading-context anchor that matches at every
// line start, so an input of N newlines yields ~N matches. Unlike "x*", this
// pattern cannot stream incrementally on RE2; it is matched in one bounded
// FindAll pass whose limit is the cap, so the cap is enforced without first
// materializing every line-start match.
func TestAnalyzeStringMultilineAnchorIsCapped(t *testing.T) {
	t.Parallel()

	stylesheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:output method="xml" omit-xml-declaration="yes"/>` +
		`<xsl:template match="/"><out>` +
		`<xsl:analyze-string select="string(.)" regex="^" flags="m">` +
		`<xsl:matching-substring><m/></xsl:matching-substring>` +
		`<xsl:non-matching-substring><n><xsl:value-of select="."/></n></xsl:non-matching-substring>` +
		`</xsl:analyze-string>` +
		`</out></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	// A long run of newlines: each line start is a (zero-length) match, so the
	// match count grows with the input far past the cap below.
	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc>`+strings.Repeat("\n", 5000)+`</doc>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).
		MaxResourceBytes(1000).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"a multiline-anchor xsl:analyze-string over a large input must honor the resource cap")
	require.ErrorIs(t, err, xslt3.ErrDynamicError,
		"the analyze-string cap breach is a runtime (dynamic) error")
}

// An xsl:analyze-string resource-cap breach must be a catchable dynamic error
// carrying a concrete, non-empty $err:code (XTDE1140), so an xsl:catch can match
// on it. A breach reported with an empty code would leave $err:code empty and
// defeat code-specific catch matching.
func TestAnalyzeStringCapBreachCarriesCatchableCode(t *testing.T) {
	t.Parallel()

	stylesheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0"` +
		` xmlns:xsl="http://www.w3.org/1999/XSL/Transform"` +
		` xmlns:err="http://www.w3.org/2005/xqt-errors">` +
		`<xsl:output method="xml" omit-xml-declaration="yes"/>` +
		`<xsl:template match="/"><out>` +
		`<xsl:try>` +
		`<xsl:analyze-string select="string(.)" regex="x*">` +
		`<xsl:matching-substring><m/></xsl:matching-substring>` +
		`<xsl:non-matching-substring><n/></xsl:non-matching-substring>` +
		`</xsl:analyze-string>` +
		`<xsl:catch><code><xsl:value-of select="local-name-from-QName($err:code)"/></code></xsl:catch>` +
		`</xsl:try>` +
		`</out></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc>`+strings.Repeat("a", 5000)+`</doc>`))
	require.NoError(t, err)

	result, err := ss.Transform(source).
		MaxResourceBytes(1000).
		Serialize(t.Context())
	require.NoError(t, err, "the analyze-string cap breach must be catchable, not propagate")
	require.Contains(t, result, "<code>XTDE1140</code>",
		"$err:code must carry the concrete XTDE1140 code so xsl:catch can match it")
}

// A cancelled context is honored promptly by xsl:analyze-string, well short of
// running its per-segment loop to completion.
func TestAnalyzeStringHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(analyzeStringStylesheet("x*")))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc>`+strings.Repeat("a", 5000)+`</doc>`))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = ss.Transform(source).Serialize(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled,
		"a cancelled context must be honored during xsl:analyze-string")
}

// nilledFwdSchema declares <doc>'s <n> child as a NILLABLE xs:integer, so a
// source <n xsi:nil="true"/> validates as a nilled element carrying the xs:integer
// annotation. The source references the schema through xsi:schemaLocation (NOT
// xsl:import-schema), so validation runs at the source-document level.
const nilledFwdSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:nilfwd"
           xmlns:t="urn:nilfwd"
           elementFormDefault="qualified">
  <xs:element name="doc" type="t:docType"/>
  <xs:complexType name="docType">
    <xs:sequence>
      <xs:element name="n" type="xs:integer" nillable="true"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

// TestNilledForwardedToXPath3 verifies that the PSVI nilled property tracked by
// xslt3 during source validation is forwarded into the xpath3 evaluator, so the
// xpath3-owned semantics — fn:data of a nilled element is the empty sequence,
// and an element(name, type) instance-of test excludes a nilled element while
// element(name, type?) still matches it — are nilled-aware inside an
// xslt3-evaluated expression. fn:nilled itself uses the xslt3 override and is
// asserted as a sanity control. xsl:strip-space is active to force the
// validated-copy source path (which records nilled flags on the copy the
// transform navigates).
func TestNilledForwardedToXPath3(t *testing.T) {
	t.Parallel()

	const schemaLoc = "mem:/nilfwd/schema.xsd"

	const stylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:nilfwd"
  version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="string-join((
      if (nilled(/t:doc/t:n)) then 'nil' else 'notnil',
      if (/t:doc/t:n instance of element(t:n, xs:integer)) then 'is-int' else 'not-int',
      if (/t:doc/t:n instance of element(t:n, xs:integer?)) then 'q-is-int' else 'q-not-int',
      string(count(data(/t:doc/t:n)))
    ), '|')"/>
  </xsl:template>
</xsl:stylesheet>`

	const src = `<?xml version="1.0"?>
<doc xmlns="urn:nilfwd"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:schemaLocation="urn:nilfwd mem:/nilfwd/schema.xsd">
  <n xsi:nil="true"/>
</doc>`

	ctx := t.Context()
	ssDoc, err := helium.NewParser().Parse(ctx, []byte(stylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
	require.NoError(t, err)

	resolver := &exactRuntimeURIResolver{files: map[string]string{schemaLoc: nilledFwdSchema}}
	source, err := helium.NewParser().Parse(ctx, []byte(src))
	require.NoError(t, err)

	out, err := ss.Transform(source).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err)
	require.True(t, resolver.askedFor(schemaLoc), "resolver must be asked for the source schema")

	// nil       : fn:nilled true (control, via xslt3 override)
	// not-int   : nilled element does NOT match element(t:n, xs:integer) — the forwarded fix
	// q-is-int  : nilled element DOES match element(t:n, xs:integer?)
	// 0         : data() of a nilled element is the empty sequence
	require.Equal(t, "nil|not-int|q-is-int|0", out)
}

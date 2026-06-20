package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestGlobalContextItemDeclSiteNSNotPollutedByInclude verifies that the
// declaration-site namespace context of an xsl:global-context-item is derived
// from the element's own in-scope namespaces — not from the mutable compiler
// binding map, which an earlier xsl:include module can pollute by redeclaring
// the same prefix to a different URI. The main stylesheet binds p to urn:right;
// an included module binds p to urn:wrong. The global-context-item's @as type
// must resolve p against the main stylesheet (urn:right).
func TestGlobalContextItemDeclSiteNSNotPollutedByInclude(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"
	const includeURI = "mem:/stylesheets/inc.xsl"

	// Included module redeclares p to a *different* URI. Processing it pollutes
	// the compiler's mutable nsBindings if they leak across modules.
	included := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:p="urn:wrong">
  <xsl:template name="noop"/>
</xsl:stylesheet>`

	// Main stylesheet binds p to urn:right and declares the global-context-item.
	// The xsl:include comes BEFORE the declaration so the wrong binding would be
	// in c.nsBindings by the time the declaration is compiled if it leaked.
	main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:p="urn:right">
  <xsl:include href="inc.xsl"/>
  <xsl:global-context-item as="document-node(element(p:root))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="name(/*)"/></out>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()
	resolver := fileMapResolver{files: map[string]string{
		includeURI: included,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(main))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err)

	// A root in urn:right must be accepted: p resolves to the main URI.
	right, err := helium.NewParser().Parse(ctx, []byte(`<root xmlns="urn:right"/>`))
	require.NoError(t, err)
	out, err := xslt3.TransformString(ctx, right, ss)
	require.NoError(t, err, "p must resolve to the main stylesheet URI (urn:right), not the included module's urn:wrong")
	require.Contains(t, out, "root")

	// A root in urn:wrong must be rejected.
	wrong, err := helium.NewParser().Parse(ctx, []byte(`<root xmlns="urn:wrong"/>`))
	require.NoError(t, err)
	_, err = xslt3.TransformString(ctx, wrong, ss)
	require.Error(t, err, "root in the included module's urn:wrong must be rejected")
}

// TestGlobalContextItemXPathDefaultNamespaceEmptyClears verifies that an
// explicit xpath-default-namespace="" on xsl:global-context-item clears an
// inherited default element namespace: an unprefixed element test in @as then
// means a no-namespace element, regardless of the stylesheet-wide default.
func TestGlobalContextItemXPathDefaultNamespaceEmptyClears(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xpath-default-namespace="urn:inherited">
  <xsl:global-context-item xpath-default-namespace=""
    as="document-node(element(root))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="local-name(/*)"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	// A no-namespace root must be accepted: the empty xpath-default-namespace
	// clears the inherited urn:inherited default, so element(root) is {}root.
	noNS, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)
	out, err := xslt3.TransformString(t.Context(), noNS, ss)
	require.NoError(t, err, "no-namespace root must match element(root) after xpath-default-namespace=\"\" clears the inherited default")
	require.Contains(t, out, "root")

	// A root in the inherited namespace must be rejected: the default was cleared.
	inherited, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns="urn:inherited"/>`))
	require.NoError(t, err)
	_, err = xslt3.TransformString(t.Context(), inherited, ss)
	require.Error(t, err, "root in the inherited namespace must be rejected once the default is cleared")
}

// TestGlobalContextItemSchemaElementDeclSiteNS verifies that schema-element()
// in an xsl:global-context-item @as type resolves its prefix against the
// declaration-site namespace context — here p is bound only on the
// xsl:global-context-item element itself, not stylesheet-wide. The @as type
// must validate against the imported schema's {urn:right}root element.
func TestGlobalContextItemSchemaElementDeclSiteNS(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/s.xsd"

	schema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:right"
           xmlns:p="urn:right"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	// p is declared ONLY on the xsl:global-context-item element. The
	// import-schema uses a different prefix (s) for the same URI to ensure the
	// schema-element(p:root) resolution comes from the declaration site, not a
	// stylesheet-wide binding for p.
	main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="urn:right">
  <xsl:import-schema namespace="urn:right" schema-location="s.xsd"/>
  <xsl:global-context-item xmlns:p="urn:right"
    as="document-node(schema-element(p:root))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="name(/*)"/></out>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()
	resolver := fileMapResolver{files: map[string]string{
		schemaURI: schema,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(main))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "schema-element(p:root) must resolve p against the declaration site (urn:right) and find the imported {urn:right}root element")
	require.NotNil(t, ss)
}

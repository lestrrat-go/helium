package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestXMLOutputNeverEmitsXmlnsXml verifies the xslt3 XML output method never
// serializes a redundant xmlns:xml="http://www.w3.org/XML/1998/namespace"
// declaration on a literal result element — the "xml" prefix is predefined by
// the Namespaces in XML spec and bound implicitly everywhere. The regression
// surfaced on literal result elements produced by an imported module's named
// template. A genuine namespace declaration (xmlns:foo) and an xml:lang
// attribute must still serialize normally.
func TestXMLOutputNeverEmitsXmlnsXml(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// The imported module's named template emits a literal result element. In
	// the buggy path the LRE picked up a spurious xml-prefix namespace
	// declaration from the module's in-scope bindings.
	const layoutModule = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">
  <xsl:template name="page-header">
    <header xmlns:foo="urn:example:foo" xml:lang="en">
      <h1>Title</h1>
    </header>
  </xsl:template>
</xsl:stylesheet>`

	const main = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:import href="layout.xsl"/>
  <xsl:output method="xml" indent="no" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <page>
      <xsl:call-template name="page-header"/>
    </page>
  </xsl:template>
</xsl:stylesheet>`

	resolver := &recordingCompileResolver{files: map[string][]byte{
		"mem:/styles/layout.xsl": []byte(layoutModule),
	}}

	doc, err := helium.NewParser().Parse(ctx, []byte(main))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI("mem:/styles/main.xsl").
		URIResolver(resolver).
		Compile(ctx, doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	out, err := xslt3.TransformString(ctx, source, ss)
	require.NoError(t, err)

	require.NotContains(t, out, "xmlns:xml=",
		"redundant xmlns:xml declaration must never be serialized; got %q", out)
	require.Contains(t, out, `xmlns:foo="urn:example:foo"`,
		"a genuine namespace declaration must still serialize; got %q", out)
	require.Contains(t, out, `xml:lang="en"`,
		"an xml:lang attribute must still serialize; got %q", out)
	require.True(t, strings.Contains(out, "<header"), "got %q", out)
}

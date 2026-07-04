package html_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"

	"github.com/stretchr/testify/require"
)

// TestWriteNeverEmitsXmlnsXml verifies the HTML serializer never emits a
// redundant xmlns:xml="http://www.w3.org/XML/1998/namespace" declaration. The
// "xml" prefix is predefined by the Namespaces in XML spec and bound
// implicitly everywhere, so declaring it is redundant and non-canonical. A
// real namespace declaration (xmlns:foo) and an xml:lang attribute must still
// serialize normally.
func TestWriteNeverEmitsXmlnsXml(t *testing.T) {
	const xmlNS = "http://www.w3.org/XML/1998/namespace"

	doc := helium.NewHTMLDocument()
	root := doc.CreateElement("html")
	require.NoError(t, doc.SetDocumentElement(root))

	// A non-HTML-table element that carries a spurious xml namespace decl node
	// plus a genuine namespace decl and an xml:lang attribute.
	child := doc.CreateElement("header")
	require.NoError(t, child.DeclareNamespace("xml", xmlNS))
	require.NoError(t, child.DeclareNamespace("foo", "urn:example:foo"))
	langNS := helium.NewNamespace("xml", xmlNS)
	_, err := child.SetAttributeNS("lang", "en", langNS)
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))

	var buf bytes.Buffer
	err = html.NewWriter().PreserveCase(true).Format(false).
		DefaultDTD(false).WriteTo(&buf, doc)
	require.NoError(t, err)
	out := buf.String()

	require.NotContains(t, out, "xmlns:xml=",
		"redundant xmlns:xml declaration must never be serialized; got %q", out)
	require.Contains(t, out, `xmlns:foo="urn:example:foo"`,
		"a genuine namespace declaration must still serialize; got %q", out)
	require.Contains(t, out, `xml:lang="en"`,
		"an xml:lang attribute must still serialize; got %q", out)
	// Sanity: the redundant declaration was actually present as a node.
	require.True(t, strings.Contains(out, "<header"), "got %q", out)
}

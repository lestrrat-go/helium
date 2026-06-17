package html_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"

	"github.com/stretchr/testify/require"
)

// TestWriteRejectsInvalidElementName ensures the HTML serializer refuses to
// write an element whose name is not a valid XML/HTML name. Without
// validation, a name containing a space or '>' is written verbatim into the
// tag, producing malformed or injected markup.
func TestWriteRejectsInvalidElementName(t *testing.T) {
	for _, name := range []string{"a b", "a>x", `a"x`, "a<x", ""} {
		doc := helium.NewHTMLDocument()
		root := doc.CreateElement(name)
		require.NoError(t, doc.SetDocumentElement(root))

		out, err := html.WriteString(doc)
		require.Error(t, err, "serializing element name %q must error, got output %q", name, out)
	}
}

// TestWriteRejectsInvalidAttributeName ensures the HTML serializer refuses to
// write an attribute whose name is not a valid XML/HTML name, since such a
// name (with a space or quote) would otherwise inject markup into the tag.
func TestWriteRejectsInvalidAttributeName(t *testing.T) {
	for _, name := range []string{"a b", `a"x`, "a=x", "a>x"} {
		doc := helium.NewHTMLDocument()
		root := doc.CreateElement("div")
		// Use the namespaced setter to bypass the colon check; these names
		// contain no colon but are still invalid.
		require.NoError(t, root.SetLiteralAttribute(name, "v"))
		require.NoError(t, doc.SetDocumentElement(root))

		out, err := html.WriteString(doc)
		require.Error(t, err, "serializing attribute name %q must error, got output %q", name, out)
	}
}

// TestWriteRejectsInvalidBooleanAttributeName covers the boolean-attribute
// path (attribute with no value child).
func TestWriteRejectsInvalidBooleanAttributeName(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root := doc.CreateElement("input")
	require.NoError(t, root.SetBooleanAttribute("a b"))
	require.NoError(t, doc.SetDocumentElement(root))

	out, err := html.WriteString(doc)
	require.Error(t, err, "serializing boolean attribute with space must error, got output %q", out)
}

// TestWriteAcceptsValidNamespacedName confirms that valid names containing a
// single colon (namespaced names used in XSLT HTML5 output) are still
// accepted.
func TestWriteAcceptsValidNamespacedName(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root := doc.CreateElement("html")
	child := doc.CreateElement("svg:rect")
	require.NoError(t, root.AddChild(child))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err := html.NewWriter().PreserveCase(true).Format(false).
		DefaultDTD(false).WriteTo(&buf, doc)
	require.NoError(t, err)
	require.True(t, strings.Contains(buf.String(), "svg:rect"), "got %q", buf.String())
}

// TestWriteAcceptsHTMLLooseName confirms that names HTML tolerates but XML
// does not (e.g. attribute names like "gentus?.?" the HTML parser produces
// from malformed markup) are still serialized verbatim, matching libxml2.
func TestWriteAcceptsHTMLLooseName(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root := doc.CreateElement("img")
	require.NoError(t, root.SetBooleanAttribute("gentus?.?"))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err := html.NewWriter().DefaultDTD(false).Format(false).WriteTo(&buf, doc)
	require.NoError(t, err)
	require.True(t, strings.Contains(buf.String(), "gentus?.?"), "got %q", buf.String())
}

// TestWriteRejectsInjectedNamespacePrefix ensures the HTML serializer refuses
// to write a namespace declaration whose prefix is unsafe. DeclareNamespace
// accepts an arbitrary prefix, so without validation a prefix like
// `p injected="1` would be written verbatim into the xmlns: attribute name
// under PreserveCase(true), producing an injected attribute in the tag.
func TestWriteRejectsInjectedNamespacePrefix(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root := doc.CreateElement("html")
	require.NoError(t, root.DeclareNamespace(`p injected="1`, "urn:x"))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err := html.NewWriter().PreserveCase(true).Format(false).
		DefaultDTD(false).WriteTo(&buf, doc)
	require.Error(t, err, "serializing injected namespace prefix must error, got output %q", buf.String())
	require.NotContains(t, buf.String(), `injected="1`, "injected markup must not be written")
}

// TestWriteRejectsInjectedAttributeNamespacePrefix covers the
// emitAttrNSDecls path: an attribute whose namespace prefix is unsafe must
// also be rejected rather than emitted as an xmlns: declaration.
func TestWriteRejectsInjectedAttributeNamespacePrefix(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root := doc.CreateElement("svg")
	ns, err := doc.CreateNamespace(`p injected="1`, "urn:x")
	require.NoError(t, err)
	require.NoError(t, root.SetLiteralAttributeNS("href", "v", ns))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err = html.NewWriter().PreserveCase(true).Format(false).
		DefaultDTD(false).WriteTo(&buf, doc)
	require.Error(t, err, "serializing injected attribute namespace prefix must error, got output %q", buf.String())
	require.NotContains(t, buf.String(), `injected="1`, "injected markup must not be written")
}

// TestRoundTripAmpersandInAttrName proves that '&' in an attribute name
// round-trips through parse and serialize. The HTML parser's liberal
// attribute-name rule accepts '&' (it does not terminate the tag), so the
// serializer must accept it too; rejecting it would regress parity.
func TestRoundTripAmpersandInAttrName(t *testing.T) {
	const input = `<div a&b=v></div>`
	doc, err := html.NewParser().SuppressImplied(true).Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = html.NewWriter().DefaultDTD(false).Format(false).WriteTo(&buf, doc)
	require.NoError(t, err, "serializing parsed `a&b` attribute name must not error")
	require.Contains(t, buf.String(), "a&b", "attribute name with '&' should round-trip, got %q", buf.String())
}

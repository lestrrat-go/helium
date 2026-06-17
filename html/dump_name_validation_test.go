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

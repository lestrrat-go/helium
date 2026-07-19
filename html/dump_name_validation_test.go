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
		root, err := doc.CreateElement(name)
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		out, err := html.WriteString(doc)
		require.Error(t, err, "serializing element name %q must error, got output %q", name, out)
	}
}

// TestWriteRejectsMalformedElementName ensures the HTML serializer refuses to
// write element names that libxml2 treats as malformed tag names — e.g.
// CreateElement("a?b") or CreateElement("a&b") — rather than serializing them
// verbatim as <a?b> / <a&b>. Element names use the stricter HTML tag-name
// grammar; '?' and '&' (accepted in the loose ATTRIBUTE rule) are rejected
// here.
func TestWriteRejectsMalformedElementName(t *testing.T) {
	for _, name := range []string{"a?b", "a&b", "a=b"} {
		doc := helium.NewHTMLDocument()
		root, err := doc.CreateElement(name)
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		out, err := html.WriteString(doc)
		require.Error(t, err, "serializing malformed element name %q must error, got output %q", name, out)
	}
}

// TestWriteAcceptsValidElementNames confirms the stricter element-name grammar
// does not over-reject names the parser/suite legitimately produces: ASCII
// tag-name characters (letters, digits, ':', '-', '_', '.').
func TestWriteAcceptsValidElementNames(t *testing.T) {
	for _, name := range []string{"div", "my-elem", "a.b", "x_y", "h1", "svg:rect"} {
		doc := helium.NewHTMLDocument()
		root, err := doc.CreateElement("html")
		require.NoError(t, err)

		var child *helium.Element
		if prefix, local, found := strings.Cut(name, ":"); found {
			ns, nsErr := doc.CreateNamespace(prefix, "http://www.w3.org/2000/svg")
			require.NoError(t, nsErr)
			child, err = doc.CreateElementNS(local, ns)
		} else {
			child, err = doc.CreateElement(name)
		}
		require.NoError(t, err)
		require.NoError(t, root.AddChild(child))
		require.NoError(t, doc.SetDocumentElement(root))

		var buf bytes.Buffer
		err = html.NewWriter().PreserveCase(true).Format(false).
			DefaultDTD(false).WriteTo(&buf, doc)
		require.NoError(t, err, "serializing valid element name %q must not error", name)
		require.Contains(t, buf.String(), name, "got %q", buf.String())
	}
}

// TestRoundTripLooseCharsInAttrNameNotElement proves the split: '?' and '&'
// remain valid in ATTRIBUTE names (loose rule, libxml2 round-trip parity) even
// though they are rejected in ELEMENT names.
func TestRoundTripLooseCharsInAttrNameNotElement(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root, err := doc.CreateElement("img")
	require.NoError(t, err)
	require.NoError(t, root.SetBooleanAttribute("a?b"))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err = html.NewWriter().DefaultDTD(false).Format(false).WriteTo(&buf, doc)
	require.NoError(t, err, "attribute name 'a?b' must remain valid (loose rule)")
	require.Contains(t, buf.String(), "a?b", "got %q", buf.String())
}

// TestWriteRejectsInvalidAttributeName ensures the HTML serializer refuses to
// write an attribute whose name is not a valid XML/HTML name, since such a
// name (with a space or quote) would otherwise inject markup into the tag.
func TestWriteRejectsInvalidAttributeName(t *testing.T) {
	for _, name := range []string{"a b", `a"x`, "a=x", "a>x"} {
		doc := helium.NewHTMLDocument()
		root, err := doc.CreateElement("div")
		require.NoError(t, err)
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
	root, err := doc.CreateElement("input")
	require.NoError(t, err)
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
	root, err := doc.CreateElement("html")
	require.NoError(t, err)
	svgNS, err := doc.CreateNamespace("svg", "http://www.w3.org/2000/svg")
	require.NoError(t, err)
	child, err := doc.CreateElementNS("rect", svgNS)
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err = html.NewWriter().PreserveCase(true).Format(false).
		DefaultDTD(false).WriteTo(&buf, doc)
	require.NoError(t, err)
	require.True(t, strings.Contains(buf.String(), "svg:rect"), "got %q", buf.String())
}

// TestWriteAcceptsHTMLLooseName confirms that names HTML tolerates but XML
// does not (e.g. attribute names like "gentus?.?" the HTML parser produces
// from malformed markup) are still serialized verbatim, matching libxml2.
func TestWriteAcceptsHTMLLooseName(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root, err := doc.CreateElement("img")
	require.NoError(t, err)
	require.NoError(t, root.SetBooleanAttribute("gentus?.?"))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err = html.NewWriter().DefaultDTD(false).Format(false).WriteTo(&buf, doc)
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
	root, err := doc.CreateElement("html")
	require.NoError(t, err)
	require.NoError(t, root.DeclareNamespace(`p injected="1`, "urn:x"))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err = html.NewWriter().PreserveCase(true).Format(false).
		DefaultDTD(false).WriteTo(&buf, doc)
	require.Error(t, err, "serializing injected namespace prefix must error, got output %q", buf.String())
	require.NotContains(t, buf.String(), `injected="1`, "injected markup must not be written")
}

// TestWriteRejectsInjectedAttributeNamespacePrefix covers the
// emitAttrNSDecls path: an attribute whose namespace prefix is unsafe must
// also be rejected rather than emitted as an xmlns: declaration.
func TestWriteRejectsInjectedAttributeNamespacePrefix(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root, err := doc.CreateElement("svg")
	require.NoError(t, err)
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

// TestRoundTripReplacementCharInAttrName proves that a validly-encoded U+FFFD
// (REPLACEMENT CHARACTER) in an attribute name round-trips through parse and
// serialize. The HTML parser accepts any non-terminator character, and a real
// U+FFFD does not break out of the tag, so the serializer must accept it too.
func TestRoundTripReplacementCharInAttrName(t *testing.T) {
	const input = "<div a�b=v></div>"
	doc, err := html.NewParser().SuppressImplied(true).Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = html.NewWriter().DefaultDTD(false).Format(false).WriteTo(&buf, doc)
	require.NoError(t, err, "serializing parsed U+FFFD attribute name must not error")
	require.Contains(t, buf.String(), "a�b", "attribute name with U+FFFD should round-trip, got %q", buf.String())
}

// TestWriteRejectsInvalidUTF8InAttributeName ensures that an actually-invalid
// UTF-8 byte sequence in a name is still rejected, distinct from a valid
// U+FFFD. A lone 0xFF byte is not valid UTF-8 and must not be serialized.
// PreserveCase(true) is used so the raw name reaches checkName verbatim; the
// lowercasing path would otherwise sanitize the invalid byte into a valid
// U+FFFD before validation.
func TestWriteRejectsInvalidUTF8InAttributeName(t *testing.T) {
	doc := helium.NewHTMLDocument()
	root, err := doc.CreateElement("div")
	require.NoError(t, err)
	require.NoError(t, root.SetLiteralAttribute("a\xffb", "v"))
	require.NoError(t, doc.SetDocumentElement(root))

	var buf bytes.Buffer
	err = html.NewWriter().PreserveCase(true).DefaultDTD(false).Format(false).WriteTo(&buf, doc)
	require.Error(t, err, "serializing attribute name with invalid UTF-8 must error, got output %q", buf.String())
}

package html_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"

	"github.com/stretchr/testify/require"
)

// TestWriterNormalization exercises html.Writer.Normalization: Unicode
// normalization is scoped to text-node and attribute-value character content
// (Serialization 3.1 §4). Element/attribute names are never normalized.
func TestWriterNormalization(t *testing.T) {
	const decomposed = "e\u0301" // "e" + combining acute
	const composed = "\u00e9"    // U+00E9

	doc := helium.NewHTMLDocument()
	// Element named "p" + decomposed é, with a decomposed text child.
	root := doc.CreateElement("p" + decomposed)
	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.AddChild(doc.CreateText([]byte(decomposed))))

	var buf strings.Builder
	err := html.NewWriter().DefaultDTD(false).Format(false).PreserveCase(true).
		Normalization("NFC").WriteTo(&buf, root)
	require.NoError(t, err)
	out := buf.String()

	require.Contains(t, out, "<p"+decomposed+">", "element name not normalized: %q", out)
	require.Contains(t, out, ">"+composed+"<", "text content normalized: %q", out)
	require.NotContains(t, out, "p"+composed, "name must stay decomposed: %q", out)

	// Without normalization the text stays decomposed (byte-identical default).
	var raw strings.Builder
	require.NoError(t, html.NewWriter().DefaultDTD(false).Format(false).PreserveCase(true).WriteTo(&raw, root))
	require.NotContains(t, raw.String(), composed, "no normalization by default: %q", raw.String())
}

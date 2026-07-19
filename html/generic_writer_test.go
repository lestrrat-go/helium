package html_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestGenericWriterSerializesParsedHTMLDocument verifies that the generic XML
// writer (helium.WriteString) can serialize a document produced by the HTML
// parser. The parsed document node is of HTMLDocumentNode type, so the writer
// must route it through its DocumentNode path and serialize the children rather
// than rejecting the document node's "(document)" name.
func TestGenericWriterSerializesParsedHTMLDocument(t *testing.T) {
	const src = `<!DOCTYPE html><html><head><title>t</title></head>` +
		`<body><p>Hello</p></body></html>`

	doc, err := html.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse fixture")
	require.Equal(t, helium.HTMLDocumentNode, doc.Type(), "parser must yield an HTML document node")

	out, err := helium.WriteString(doc)
	require.NoError(t, err, "the generic writer must serialize a parsed HTML document")
	require.Contains(t, out, "<title>t</title>", "the parsed children must serialize")
	require.Contains(t, out, "Hello", "the body text must serialize")
}

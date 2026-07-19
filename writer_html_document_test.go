package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestWriteHTMLDocumentNode verifies the generic XML writer serializes a
// document node of HTMLDocumentNode type (as produced by NewHTMLDocument and by
// the HTML parser) through the same DocumentNode path as an XML document, rather
// than falling to the element path and rejecting the "(document)" node name.
func TestWriteHTMLDocumentNode(t *testing.T) {
	t.Parallel()

	doc := helium.NewHTMLDocument()
	root, err := doc.CreateElement("html")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))

	body, err := doc.CreateElement("body")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(body))
	require.NoError(t, body.AddChild(doc.CreateText([]byte("hi"))))

	out, err := helium.WriteString(doc)
	require.NoError(t, err, "generic writer must serialize an HTMLDocumentNode document")

	require.True(t, strings.Contains(out, "<body>hi</body>") || strings.Contains(out, "<body>hi</body >"),
		"the HTML document's children must serialize, got %q", out)
	require.Contains(t, out, "<html>", "the document element must serialize")
}

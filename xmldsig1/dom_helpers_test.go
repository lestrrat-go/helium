package xmldsig1

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestSigAnchor_DocumentParent exercises the case where the Signature element
// is the document's root element. captureAnchor must record the Document as
// the parent (not nil) so that restore can reattach the node after a
// temporary detach during enveloped-signature processing.
func TestSigAnchor_DocumentParent(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"/>`))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)

	anchor := captureAnchor(root)
	require.NotNil(t, anchor.parent, "anchor must record the Document parent")

	helium.UnlinkNode(root)
	require.Nil(t, doc.DocumentElement(), "root should be detached")

	require.NoError(t, anchor.restore(root))
	require.Same(t, root, doc.DocumentElement(), "restore must reattach the Signature as document element")
}

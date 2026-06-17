package xmldsig1

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestCanonicalizeSubtreeKeepsNamespaceDecls is the load-bearing regression
// guard. A namespace-qualified signed subtree must canonicalize WITH its
// in-scope xmlns declarations. c14n node-set mode only emits namespaces that
// are explicitly present in the node set, so collectSubtreeNodes must include
// the in-scope namespace axis for every element. Without it the prefixed
// element names are emitted WITHOUT their xmlns:p declaration, producing
// non-W3C canonical bytes that break cross-implementation signature interop.
func TestCanonicalizeSubtreeKeepsNamespaceDecls(t *testing.T) {
	const xml = `<doc xmlns:p="urn:p"><p:target Id="x"><p:child>v</p:child></p:target></doc>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	target, err := resolveReference(doc, "#x")
	require.NoError(t, err)

	t.Run("inclusive C14N 1.0", func(t *testing.T) {
		out, err := canonicalizeSubtree(C14N10, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"canonical subtree must carry the in-scope xmlns:p declaration")
		// The prefixed element name and its namespace decl must coexist.
		require.True(t, strings.HasPrefix(string(out), `<p:target xmlns:p="urn:p"`),
			"got: %s", out)
	})

	t.Run("exclusive C14N 1.0", func(t *testing.T) {
		out, err := canonicalizeSubtree(ExcC14N10, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"exclusive canonical subtree must carry the visibly-utilized xmlns:p declaration")
	})

	t.Run("C14N 1.1", func(t *testing.T) {
		out, err := canonicalizeSubtree(C14N11URI, target, nil)
		require.NoError(t, err)
		require.Contains(t, string(out), `xmlns:p="urn:p"`,
			"C14N 1.1 canonical subtree must carry the in-scope xmlns:p declaration")
	})
}

package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestElementNamespaceMutators exercises the namespace-declaration mutators on
// an element: DeclareNamespace, AddNamespaceDecl, RemoveNamespaceByPrefix,
// SetActiveNamespace and SetNs.
func TestElementNamespaceMutators(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	e := doc.CreateElement("e")

	require.NoError(t, e.DeclareNamespace("p", "urn:p"))
	require.Len(t, e.Namespaces(), 1)

	shared := helium.NewNamespace("q", "urn:q")
	e.AddNamespaceDecl(shared)
	require.Len(t, e.Namespaces(), 2)

	require.True(t, e.RemoveNamespaceByPrefix("p"), "existing prefix removed")
	require.False(t, e.RemoveNamespaceByPrefix("absent"), "missing prefix is a no-op")
	require.Len(t, e.Namespaces(), 1)

	require.NoError(t, e.SetActiveNamespace("a", "urn:a"))
	require.Equal(t, "a", e.Prefix())
	require.Equal(t, "urn:a", e.URI())
	require.Equal(t, "a:e", e.Name())

	e.SetNs(shared)
	require.Equal(t, "q", e.Prefix())
	require.Equal(t, "urn:q", e.URI())
}

// TestEncodingDeclarations parses documents with various encoding declarations
// to exercise the encoding-switch paths.
func TestEncodingDeclarations(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`<?xml version="1.0" encoding="UTF-8"?><root>ascii</root>`,
		`<?xml version="1.0" encoding="utf-8"?><root>ascii</root>`,
		`<?xml version="1.0" encoding="US-ASCII"?><root>ascii</root>`,
	}
	for _, in := range inputs {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(in))
		require.NoError(t, err, "parse %q", in)
		require.NotNil(t, doc.DocumentElement())
	}
}

// TestEncodingIgnored verifies the IgnoreEncoding option does not break a parse
// that declares an encoding.
func TestEncodingIgnored(t *testing.T) {
	t.Parallel()

	const in = `<?xml version="1.0" encoding="ISO-8859-1"?><root>x</root>`
	doc, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(), []byte(in))
	require.NoError(t, err)
	require.NotNil(t, doc.DocumentElement())
}

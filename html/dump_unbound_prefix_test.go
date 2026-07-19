package html_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestUnboundPrefixSerialization pins the split between the two serializers for a
// colon-bearing HTML tag name. html.Parse builds <foo:bar> through
// CreateNamespace(prefix, "") — an intentional empty-URI binding — so the
// prefix "foo" is unbound. The HTML serializer emits the name verbatim and keeps
// working. The generic XML writer must instead reject the element: emitting
// "foo:bar" with no xmlns:foo produces output helium's own parser cannot reparse.
func TestUnboundPrefixSerialization(t *testing.T) {
	const input = `<div><foo:bar>inner</foo:bar></div>`

	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	elem := findElement(doc, "foo:bar")
	require.NotNil(t, elem, "parser must build the colon-named element")

	// The HTML serializer keeps working on the empty-URI-prefixed DOM.
	htmlOut, err := html.WriteString(elem)
	require.NoError(t, err, "HTML serializer must handle the unbound-prefix DOM")
	require.Contains(t, htmlOut, "foo:bar")

	// The generic XML writer refuses the same element rather than emit
	// namespace-ill-formed output.
	_, err = helium.WriteString(elem)
	require.Error(t, err, "generic writer must reject the unbound prefix")
	require.ErrorIs(t, err, helium.ErrWriterUnboundNamespacePrefix)
}

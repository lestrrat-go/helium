package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const namespaceAxisXML = `<root xmlns="urn:root" xmlns:xlink="http://www.w3.org/1999/xlink">
  <child xmlns:local="urn:local"/>
</root>`

func parseNamespaceAxisDoc(t *testing.T) *helium.Document {
	t.Helper()

	doc, err := helium.Parse(t.Context(), []byte(namespaceAxisXML))
	require.NoError(t, err)
	return doc
}

func TestNamespaceAxisStringValueAndParent(t *testing.T) {
	doc := parseNamespaceAxisDoc(t)

	result, err := evaluate(t.Context(), doc, `string(/*/namespace::xlink)`)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "http://www.w3.org/1999/xlink", s)

	nodes, err := find(t.Context(), doc, `/*/namespace::*/..`)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "root", nodes[0].Name())
}

func TestNamespaceAxisNodeIdentity(t *testing.T) {
	doc := parseNamespaceAxisDoc(t)

	result, err := evaluate(t.Context(), doc, `/*/namespace::xlink is /*/*[1]/namespace::xlink`)
	require.NoError(t, err)

	sameInherited, ok := result.IsBoolean()
	require.True(t, ok)
	require.False(t, sameInherited)

	result, err = evaluate(t.Context(), doc, `/*/namespace::xlink is /*/namespace::*[. = 'http://www.w3.org/1999/xlink']`)
	require.NoError(t, err)

	sameLogical, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, sameLogical)

	result, err = evaluate(t.Context(), doc, `generate-id(/*/namespace::xlink) eq generate-id(/*/namespace::*[. = 'http://www.w3.org/1999/xlink'])`)
	require.NoError(t, err)

	sameID, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, sameID)
}

func TestNamespaceAxisPath(t *testing.T) {
	doc := parseNamespaceAxisDoc(t)

	result, err := evaluate(t.Context(), doc, `path((//namespace::xml)[1])`)
	require.NoError(t, err)

	xmlPath, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "/Q{urn:root}root[1]/namespace::xml", xmlPath)

	result, err = evaluate(t.Context(), doc, `path((//namespace::*[name()=''])[1])`)
	require.NoError(t, err)

	defaultPath, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, `/Q{urn:root}root[1]/namespace::*[Q{http://www.w3.org/2005/xpath-functions}local-name()=""]`, defaultPath)
}

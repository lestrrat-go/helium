package helium

import (
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestGetElementDescKey verifies that an element declaration registered via
// AddElementDecl can be retrieved through GetElementDesc using the same QName.
// Registration keys decls as "name:prefix"; GetElementDesc must compose the
// lookup key the same way instead of using the raw QName.
func TestGetElementDescKey(t *testing.T) {
	t.Run("unprefixed", func(t *testing.T) {
		dtd := newDTD()
		content, err := newElementContent("", ElementContentPCDATA)
		require.NoError(t, err)
		_, err = dtd.AddElementDecl("r", enum.MixedElementType, content)
		require.NoError(t, err)

		decl, ok := dtd.GetElementDesc("r")
		require.True(t, ok, "GetElementDesc must find the registered decl")
		require.Equal(t, enum.MixedElementType, decl.decltype)
	})
	t.Run("prefixed", func(t *testing.T) {
		dtd := newDTD()
		_, err := dtd.AddElementDecl("foo:bar", enum.EmptyElementType, nil)
		require.NoError(t, err)

		decl, ok := dtd.GetElementDesc("foo:bar")
		require.True(t, ok, "GetElementDesc must find the prefixed decl by QName")
		require.Equal(t, enum.EmptyElementType, decl.decltype)
	})
}

// TestIsMixedElementWhitespace exercises the mixed-content whitespace path that
// relies on GetElementDesc: a mixed-content element must report IsMixedElement
// true so whitespace inside it is not misclassified as ignorable.
func TestIsMixedElementWhitespace(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd := newDTD()
	dtd.doc = doc
	doc.intSubset = dtd

	content, err := newElementContent("", ElementContentPCDATA)
	require.NoError(t, err)
	_, err = dtd.AddElementDecl("r", enum.MixedElementType, content)
	require.NoError(t, err)

	mixed, err := doc.IsMixedElement("r")
	require.NoError(t, err)
	require.True(t, mixed, "mixed-content element must be reported as mixed")
}

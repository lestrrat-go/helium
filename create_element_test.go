package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestCreateElementValidation(t *testing.T) {
	t.Run("plain name succeeds", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e, err := doc.CreateElement("item")
		require.NoError(t, err)
		require.NotNil(t, e)
		require.Equal(t, "item", e.Name())
	})
	t.Run("colon in name is rejected", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e, err := doc.CreateElement("p:item")
		require.Error(t, err)
		require.Nil(t, e)
		require.Contains(t, err.Error(), "CreateElementNS")
	})
	t.Run("nil receiver plain name succeeds", func(t *testing.T) {
		var doc *helium.Document
		e, err := doc.CreateElement("item")
		require.NoError(t, err)
		require.NotNil(t, e)
		require.Equal(t, "item", e.Name())
	})
	t.Run("nil receiver colon in name is rejected", func(t *testing.T) {
		var doc *helium.Document
		e, err := doc.CreateElement("p:item")
		require.Error(t, err)
		require.Nil(t, e)
	})
}

func TestCreateElementNS(t *testing.T) {
	t.Run("prefixed namespace yields prefixed Name and serialization", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		ns, err := doc.CreateNamespace("p", "http://example.com/p")
		require.NoError(t, err)

		e, err := doc.CreateElementNS("item", ns)
		require.NoError(t, err)
		require.NotNil(t, e)
		require.Equal(t, "p:item", e.Name())

		err = e.DeclareNamespace("p", "http://example.com/p")
		require.NoError(t, err)

		s, err := helium.WriteString(e)
		require.NoError(t, err)
		require.Contains(t, s, "<p:item")
		require.Contains(t, s, `xmlns:p="http://example.com/p"`)
	})
	t.Run("colon in local name is rejected", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		ns, err := doc.CreateNamespace("p", "http://example.com/p")
		require.NoError(t, err)

		e, err := doc.CreateElementNS("a:b", ns)
		require.Error(t, err)
		require.Nil(t, e)
		require.Contains(t, err.Error(), "CreateElementNS")
	})
	t.Run("nil namespace yields unqualified element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		e, err := doc.CreateElementNS("item", nil)
		require.NoError(t, err)
		require.NotNil(t, e)
		require.Equal(t, "item", e.Name())
	})
	t.Run("nil receiver succeeds for standalone element", func(t *testing.T) {
		var doc *helium.Document
		ns, err := doc.CreateNamespace("p", "http://example.com/p")
		require.NoError(t, err)

		e, err := doc.CreateElementNS("item", ns)
		require.NoError(t, err)
		require.NotNil(t, e)
		require.Equal(t, "p:item", e.Name())
	})
	t.Run("nil receiver colon in local name is rejected", func(t *testing.T) {
		var doc *helium.Document
		e, err := doc.CreateElementNS("a:b", nil)
		require.Error(t, err)
		require.Nil(t, e)
	})
}

func TestDeclareNamespaceRoundTrip(t *testing.T) {
	doc := helium.NewDefaultDocument()
	ns, err := doc.CreateNamespace("p", "urn:x")
	require.NoError(t, err)
	e, err := doc.CreateElementNS("item", ns)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(e.Name(), "p:"))
}

package helium_test

import (
	helim "github.com/lestrrat-go/helium"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetElementByID(t *testing.T) {
	t.Parallel()

	t.Run("xml:id via parser", func(t *testing.T) {
		t.Parallel()
		const input = `<?xml version="1.0"?>
<root>
  <a xml:id="first">one</a>
  <b xml:id="second">two</b>
</root>`
		p := helim.NewParser()
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		// O(1) lookup via ID table
		elem := doc.GetElementByID("first")
		require.NotNil(t, elem)
		require.Equal(t, "a", elem.LocalName())

		elem = doc.GetElementByID("second")
		require.NotNil(t, elem)
		require.Equal(t, "b", elem.LocalName())

		// Non-existent ID
		elem = doc.GetElementByID("missing")
		require.Nil(t, elem)
	})

	t.Run("DTD-declared ID via parser", func(t *testing.T) {
		t.Parallel()
		const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (item*)>
  <!ELEMENT item (#PCDATA)>
  <!ATTLIST item eid ID #IMPLIED>
]>
<root>
  <item eid="x1">alpha</item>
  <item eid="x2">beta</item>
</root>`
		p := helim.NewParser()
		p.SetOption(helim.ParseDTDLoad | helim.ParseDTDAttr)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		elem := doc.GetElementByID("x1")
		require.NotNil(t, elem)
		require.Equal(t, "item", elem.LocalName())

		elem = doc.GetElementByID("x2")
		require.NotNil(t, elem)
		require.Equal(t, "item", elem.LocalName())
	})

	t.Run("fallback tree walk for programmatic documents", func(t *testing.T) {
		t.Parallel()
		// Documents built without parsing have no ID table,
		// so GetElementByID falls back to O(n) tree walk.
		doc := helim.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		ns := helim.NewNamespace("xml", helim.XMLNamespace)
		require.NoError(t, child.SetAttributeNS("id", "myid", ns))
		require.NoError(t, root.AddChild(child))

		elem := doc.GetElementByID("myid")
		require.NotNil(t, elem)
		require.Equal(t, "child", elem.LocalName())

		elem = doc.GetElementByID("missing")
		require.Nil(t, elem)
	})

	t.Run("ID table populated on parse", func(t *testing.T) {
		t.Parallel()
		const input = `<?xml version="1.0"?>
<root xml:id="r">
  <child xml:id="c"/>
</root>`
		p := helim.NewParser()
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		require.NotNil(t, doc.GetElementByID("r"))
		require.NotNil(t, doc.GetElementByID("c"))
	})
}

func TestDocProperties(t *testing.T) {
	t.Parallel()

	t.Run("new default document is user-built", func(t *testing.T) {
		t.Parallel()
		doc := helim.NewDefaultDocument()
		require.True(t, doc.HasProperty(helim.DocUserBuilt))
	})

	t.Run("HasProperty requires all requested bits", func(t *testing.T) {
		t.Parallel()
		doc := helim.NewDocument("1.0", "", helim.StandaloneImplicitNo)
		doc.SetProperties(helim.DocWellFormed | helim.DocXInclude)

		require.True(t, doc.HasProperty(helim.DocWellFormed))
		require.True(t, doc.HasProperty(helim.DocXInclude))
		require.True(t, doc.HasProperty(helim.DocWellFormed|helim.DocXInclude))
		require.False(t, doc.HasProperty(helim.DocWellFormed|helim.DocDTDValid))
	})
}

package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetElementByID(t *testing.T) {
	t.Run("xml:id via parser", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<root>
  <a xml:id="first">one</a>
  <b xml:id="second">two</b>
</root>`
		p := NewParser()
		doc, err := p.Parse([]byte(input))
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
		p := NewParser()
		p.SetOption(ParseDTDLoad | ParseDTDAttr)
		doc, err := p.Parse([]byte(input))
		require.NoError(t, err)

		elem := doc.GetElementByID("x1")
		require.NotNil(t, elem)
		require.Equal(t, "item", elem.LocalName())

		elem = doc.GetElementByID("x2")
		require.NotNil(t, elem)
		require.Equal(t, "item", elem.LocalName())
	})

	t.Run("fallback tree walk for programmatic documents", func(t *testing.T) {
		// Documents built without parsing have no ID table,
		// so GetElementByID falls back to O(n) tree walk.
		doc := NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		ns := NewNamespace("xml", XMLNamespace)
		require.NoError(t, child.SetAttributeNS("id", "myid", ns))
		require.NoError(t, root.AddChild(child))

		elem := doc.GetElementByID("myid")
		require.NotNil(t, elem)
		require.Equal(t, "child", elem.LocalName())

		elem = doc.GetElementByID("missing")
		require.Nil(t, elem)
	})

	t.Run("ID table populated on parse", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<root xml:id="r">
  <child xml:id="c"/>
</root>`
		p := NewParser()
		doc, err := p.Parse([]byte(input))
		require.NoError(t, err)

		// Verify the ID table exists and is populated
		require.NotNil(t, doc.ids)
		require.Len(t, doc.ids, 2)
	})
}

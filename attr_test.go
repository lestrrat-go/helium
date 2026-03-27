package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestCreateAttribute(t *testing.T) {
	t.Parallel()

	t.Run("rejects colon in name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()

		// A colon in the name parameter is invalid — the name must be an NCName.
		// Callers should use CreateAttribute(localName, value, ns) with a
		// proper Namespace object instead of passing a QName.
		_, err := doc.CreateAttribute("xml:base", "http://example.com", nil)
		require.Error(t, err)

		// Passing a proper local name should succeed.
		ns := helium.NewNamespace("xml", lexicon.NamespaceXML)
		attr, err := doc.CreateAttribute("base", "http://example.com", ns)
		require.NoError(t, err)
		require.Equal(t, "base", attr.LocalName())
		require.Equal(t, "xml:base", attr.Name())
	})
}

func TestSetAttribute(t *testing.T) {
	t.Parallel()

	t.Run("rejects colon in name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		elem := doc.CreateElement("root")

		// A colon in the name parameter is invalid — callers should use
		// SetAttributeNS with a proper Namespace object.
		_, err := elem.SetAttribute("xml:space", "preserve")
		require.Error(t, err)

		// Passing a proper local name should succeed.
		_, err = elem.SetAttribute("id", "123")
		require.NoError(t, err)
	})

	t.Run("SetLiteralAttribute rejects colon", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		elem := doc.CreateElement("root")

		// A colon in the name parameter is invalid.
		err := elem.SetLiteralAttribute("xml:lang", "en")
		require.Error(t, err)

		// Passing a proper local name should succeed.
		err = elem.SetLiteralAttribute("lang", "en")
		require.NoError(t, err)
	})
}

func TestAttributeAType(t *testing.T) {
	t.Run("explicit attributes carry atype", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE person [
  <!ELEMENT person EMPTY>
  <!ATTLIST person
    name CDATA #REQUIRED
    id ID #REQUIRED
    ref IDREF #IMPLIED
    refs IDREFS #IMPLIED
    tok NMTOKEN #IMPLIED
    toks NMTOKENS #IMPLIED
  >
]>
<person name="Alice" id="p1" ref="p1" refs="p1" tok="abc" toks="abc def"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		expected := map[string]enum.AttributeType{
			"name": enum.AttrCDATA,
			"id":   enum.AttrID,
			"ref":  enum.AttrIDRef,
			"refs": enum.AttrIDRefs,
			"tok":  enum.AttrNmtoken,
			"toks": enum.AttrNmtokens,
		}

		for _, attr := range root.Attributes() {
			want, ok := expected[attr.LocalName()]
			require.True(t, ok, "unexpected attribute %s", attr.LocalName())
			require.Equal(t, want, attr.AType(), "attribute %s should have type %d", attr.LocalName(), want)
		}
	})

	t.Run("default attributes carry atype", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE person [
  <!ELEMENT person EMPTY>
  <!ATTLIST person
    name CDATA "unknown"
    role NMTOKEN "admin"
  >
]>
<person/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		expected := map[string]enum.AttributeType{
			"name": enum.AttrCDATA,
			"role": enum.AttrNmtoken,
		}

		attrs := root.Attributes()
		require.Len(t, attrs, len(expected))
		for _, attr := range attrs {
			want, ok := expected[attr.LocalName()]
			require.True(t, ok, "unexpected attribute %s", attr.LocalName())
			require.Equal(t, want, attr.AType(), "default attribute %s should have type %d", attr.LocalName(), want)
		}
	})

	t.Run("attributes without DTD have AttrInvalid", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<root attr="value"/>`

		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		attrs := root.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, enum.AttrInvalid, attrs[0].AType())
	})

	t.Run("enumeration attributes carry atype", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) "red">
]>
<root color="green"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		attrs := root.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, enum.AttrEnumeration, attrs[0].AType())
	})
}

func TestGetAttribute(t *testing.T) {
	t.Parallel()

	t.Run("by local name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("id", "123")
		require.NoError(t, err)
		_, err = e.SetAttribute("class", "main")
		require.NoError(t, err)

		val, ok := e.GetAttribute("id")
		require.True(t, ok)
		require.Equal(t, "123", val)

		val, ok = e.GetAttribute("class")
		require.True(t, ok)
		require.Equal(t, "main", val)

		_, ok = e.GetAttribute("missing")
		require.False(t, ok)
	})

	t.Run("by namespace", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns := helium.NewNamespace("x", "http://example.com")
		_, err := e.SetAttributeNS("attr", "val", ns)
		require.NoError(t, err)

		val, ok := e.GetAttributeNS("attr", "http://example.com")
		require.True(t, ok)
		require.Equal(t, "val", val)

		_, ok = e.GetAttributeNS("attr", "http://other.com")
		require.False(t, ok)

		_, ok = e.GetAttributeNS("missing", "http://example.com")
		require.False(t, ok)
	})
}

func TestHasAttribute(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e := doc.CreateElement("root")
	_, err := e.SetAttribute("id", "123")
	require.NoError(t, err)

	require.True(t, e.HasAttribute("id"))
	require.False(t, e.HasAttribute("missing"))
}

func TestFindAttribute(t *testing.T) {
	t.Parallel()

	t.Run("by predicates", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("id", "123")
		require.NoError(t, err)

		ns := helium.NewNamespace("x", "http://example.com")
		_, err = e.SetAttributeNS("attr", "val", ns)
		require.NoError(t, err)

		attr, ok := e.FindAttribute(helium.QNamePredicate("id"))
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "123", attr.Value())

		attr, ok = e.FindAttribute(helium.QNamePredicate("x:attr"))
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "val", attr.Value())

		attr, ok = e.FindAttribute(helium.LocalNamePredicate("attr"))
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "x:attr", attr.Name())

		attr, ok = e.FindAttribute(helium.NSPredicate{Local: "attr", NamespaceURI: "http://example.com"})
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "x:attr", attr.Name())
	})

	t.Run("nil predicate", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("id", "123")
		require.NoError(t, err)

		var pred helium.AttributePredicate
		attr, ok := e.FindAttribute(pred)
		require.False(t, ok)
		require.Nil(t, attr)
	})
}

func TestGetAttributeNodeNS(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e := doc.CreateElement("root")

	ns := helium.NewNamespace("x", "http://example.com")
	_, err := e.SetAttributeNS("attr", "val", ns)
	require.NoError(t, err)

	attr := e.GetAttributeNodeNS("attr", "http://example.com")
	require.NotNil(t, attr)
	require.Equal(t, "attr", attr.LocalName())
	require.Equal(t, "val", attr.Value())
	require.Equal(t, "http://example.com", attr.URI())

	attr = e.GetAttributeNodeNS("attr", "http://other.com")
	require.Nil(t, attr)

	attr = e.GetAttributeNodeNS("missing", "http://example.com")
	require.Nil(t, attr)
}

func TestRemoveAttribute(t *testing.T) {
	t.Parallel()

	t.Run("by local name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("a", "1")
		require.NoError(t, err)
		_, err = e.SetAttribute("b", "2")
		require.NoError(t, err)
		_, err = e.SetAttribute("c", "3")
		require.NoError(t, err)

		// Remove middle
		ok := e.RemoveAttribute("b")
		require.True(t, ok)
		require.False(t, e.HasAttribute("b"))
		require.True(t, e.HasAttribute("a"))
		require.True(t, e.HasAttribute("c"))

		// Remove first
		ok = e.RemoveAttribute("a")
		require.True(t, ok)
		require.False(t, e.HasAttribute("a"))
		require.True(t, e.HasAttribute("c"))

		// Remove last (only remaining)
		ok = e.RemoveAttribute("c")
		require.True(t, ok)
		require.Equal(t, 0, len(e.Attributes()))

		// Remove non-existent
		ok = e.RemoveAttribute("missing")
		require.False(t, ok)
	})

	t.Run("by namespace", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns := helium.NewNamespace("x", "http://example.com")
		_, err := e.SetAttributeNS("attr", "val", ns)
		require.NoError(t, err)

		ok := e.RemoveAttributeNS("attr", "http://example.com")
		require.True(t, ok)
		require.Equal(t, 0, len(e.Attributes()))

		ok = e.RemoveAttributeNS("attr", "http://example.com")
		require.False(t, ok)
	})
}

func TestForEachAttribute(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e := doc.CreateElement("root")

	_, err := e.SetAttribute("a", "1")
	require.NoError(t, err)
	_, err = e.SetAttribute("b", "2")
	require.NoError(t, err)
	_, err = e.SetAttribute("c", "3")
	require.NoError(t, err)

	expected := e.Attributes()
	var iterated []*helium.Attribute
	e.ForEachAttribute(func(attr *helium.Attribute) bool {
		iterated = append(iterated, attr)
		return true
	})
	require.Equal(t, expected, iterated)

	var stopped []*helium.Attribute
	e.ForEachAttribute(func(attr *helium.Attribute) bool {
		stopped = append(stopped, attr)
		return len(stopped) < 2
	})
	require.Equal(t, expected[:2], stopped)
}

package helium

import (
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

func TestAttributeAType(t *testing.T) {
	t.Run("explicit attributes carry atype", func(t *testing.T) {
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

		p := NewParser()
		p.SetOption(ParseDTDAttr)
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
		xml := `<?xml version="1.0"?>
<!DOCTYPE person [
  <!ELEMENT person EMPTY>
  <!ATTLIST person
    name CDATA "unknown"
    role NMTOKEN "admin"
  >
]>
<person/>`

		p := NewParser()
		p.SetOption(ParseDTDAttr)
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
		xml := `<?xml version="1.0"?>
<root attr="value"/>`

		p := NewParser()
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		attrs := root.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, enum.AttrInvalid, attrs[0].AType())
	})

	t.Run("enumeration attributes carry atype", func(t *testing.T) {
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) "red">
]>
<root color="green"/>`

		p := NewParser()
		p.SetOption(ParseDTDAttr)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		attrs := root.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, enum.AttrEnumeration, attrs[0].AType())
	})
}

package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInlineAttributeSimpleType checks that an attribute use declared with an
// inline <xs:simpleType> restriction has its value validated against the
// anonymous type, rather than the restriction being silently ignored.
func TestInlineAttributeSimpleType(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a">
        <xs:simpleType>
          <xs:restriction base="xs:int"/>
        </xs:simpleType>
      </xs:attribute>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("accepts valid value", func(t *testing.T) {
		require.NoError(t, compileAndValidate(t, schemaXML, `<root a="42"/>`, nil))
	})

	t.Run("rejects invalid value", func(t *testing.T) {
		var out string
		err := compileAndValidate(t, schemaXML, `<root a="not-int"/>`, &out)
		require.Error(t, err)
		require.Contains(t, out, "is not valid for the type of attribute 'a'")
	})
}

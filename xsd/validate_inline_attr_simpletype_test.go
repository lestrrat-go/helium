package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInlineAttributeSimpleType checks that an attribute use declared with an
// inline anonymous <xs:simpleType> restriction has its value validated against
// the anonymous type, rather than the restriction being silently ignored. This
// covers all three paths that reach an attribute use:
//   - a direct local <xs:attribute name="a"> with an inline simpleType
//   - a local <xs:attribute ref="a"/> referencing a global attribute whose
//     type is an inline anonymous simpleType
//   - an <xs:anyAttribute processContents="strict"> matching such a global
//     attribute
func TestInlineAttributeSimpleType(t *testing.T) {
	t.Parallel()

	t.Run("local inline simpleType", func(t *testing.T) {
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
	})

	t.Run("ref to global attr with inline simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a">
    <xs:simpleType>
      <xs:restriction base="xs:int"/>
    </xs:simpleType>
  </xs:attribute>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="a"/>
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
	})

	t.Run("strict wildcard against global attr with inline simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a">
    <xs:simpleType>
      <xs:restriction base="xs:int"/>
    </xs:simpleType>
  </xs:attribute>
  <xs:element name="root">
    <xs:complexType>
      <xs:anyAttribute processContents="strict"/>
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
			require.Contains(t, out, "is not a valid value of the atomic type 'xs:int'")
		})
	})
}

package xsd_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// TestAttributeChildrenGrammar covers the schema-representation content model of
// a non-reference <xs:attribute> (XSD Structures §3.2.2): (annotation?, simpleType?).
// Version-independent (enforced in the default XSD 1.0 compiler). Mirrors W3C
// msMeta/Attribute_w3c.xml attI005/attI006/attP001/attP002/attQ006 cases.
func TestAttributeChildrenGrammar(t *testing.T) {
	const main = "test.xsd"
	compile := func(t *testing.T, schema string) string {
		t.Helper()
		fsys := fstest.MapFS{main: &fstest.MapFile{Data: []byte(schema)}}
		return compileFSErrors(t, fsys, main)
	}

	t.Run("nested xs:attribute child is rejected (attP001)", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="att">
    <xs:attribute name="att1" type="xs:string"/>
  </xs:attribute>
</xs:schema>`
		require.Contains(t, compile(t, schema), "is not allowed as a child of an attribute declaration")
	})

	t.Run("xs:element child is rejected (attP002)", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="att">
    <xs:element name="elem"/>
  </xs:attribute>
</xs:schema>`
		require.Contains(t, compile(t, schema), "is not allowed as a child of an attribute declaration")
	})

	t.Run("xs:complexType child is rejected (attQ006)", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="internationalPrice">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:decimal">
          <xs:attribute name="currency">
            <xs:complexType name="foo">
              <xs:sequence>
                <xs:element name="e"/>
              </xs:sequence>
            </xs:complexType>
          </xs:attribute>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compile(t, schema), "is not allowed as a child of an attribute declaration")
	})

	t.Run("annotation after simpleType is rejected (attI005)", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:attribute name="att1">
      <xs:simpleType>
        <xs:restriction base="xs:string">
          <xs:enumeration value="AK"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:annotation/>
    </xs:attribute>
  </xs:complexType>
</xs:schema>`
		require.Contains(t, compile(t, schema), "The annotation must appear before the simpleType")
	})

	t.Run("two simpleType children are rejected (attI006)", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:attribute name="att1">
      <xs:simpleType>
        <xs:restriction base="xs:string">
          <xs:enumeration value="AK"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:simpleType>
        <xs:restriction base="xs:string">
          <xs:enumeration value="AL"/>
        </xs:restriction>
      </xs:simpleType>
    </xs:attribute>
  </xs:complexType>
</xs:schema>`
		require.Contains(t, compile(t, schema), "must not have more than one simpleType")
	})

	t.Run("valid annotation then simpleType compiles", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:attribute name="att1">
      <xs:annotation><xs:documentation>doc</xs:documentation></xs:annotation>
      <xs:simpleType>
        <xs:restriction base="xs:string">
          <xs:enumeration value="AK"/>
        </xs:restriction>
      </xs:simpleType>
    </xs:attribute>
  </xs:complexType>
</xs:schema>`
		require.Empty(t, compile(t, schema))
	})

	t.Run("valid plain typed attribute compiles", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="foo" type="xs:string"/>
  <xs:complexType name="t">
    <xs:attribute name="bar" type="xs:string"/>
    <xs:attribute name="baz"/>
  </xs:complexType>
</xs:schema>`
		require.Empty(t, compile(t, schema))
	})
}

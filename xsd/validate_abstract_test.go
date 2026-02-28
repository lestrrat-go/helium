package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestAbstractTypeValidation(t *testing.T) {
	t.Run("abstract complex type rejected", func(t *testing.T) {
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType" abstract="true">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`

		instanceXML := `<root><value>hello</value></root>`

		schemaDOC, err := helium.Parse([]byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(schemaDOC)
		require.NoError(t, err)

		doc, err := helium.Parse([]byte(instanceXML))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "The type definition is abstract.")
		require.Contains(t, result, "fails to validate")
	})

	t.Run("concrete derived type via xsi:type accepted", func(t *testing.T) {
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="http://example.com" xmlns:tns="http://example.com"
  elementFormDefault="qualified">
  <xs:complexType name="baseType" abstract="true">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="concreteType">
    <xs:complexContent>
      <xs:extension base="tns:baseType"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="tns:baseType"/>
</xs:schema>`

		instanceXML := `<root xmlns="http://example.com"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:tns="http://example.com"
  xsi:type="tns:concreteType">
  <value>hello</value>
</root>`

		schemaDOC, err := helium.Parse([]byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(schemaDOC)
		require.NoError(t, err)

		doc, err := helium.Parse([]byte(instanceXML))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.Contains(t, result, "validates")
		require.NotContains(t, result, "fails to validate")
	})

	t.Run("non-abstract type accepted", func(t *testing.T) {
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="myType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="myType"/>
</xs:schema>`

		instanceXML := `<root><value>hello</value></root>`

		schemaDOC, err := helium.Parse([]byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(schemaDOC)
		require.NoError(t, err)

		doc, err := helium.Parse([]byte(instanceXML))
		require.NoError(t, err)

		result := xsd.Validate(doc, schema)
		require.True(t, strings.Contains(result, "validates") && !strings.Contains(result, "fails to validate"))
	})
}

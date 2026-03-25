package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestAbstractTypeValidation(t *testing.T) {
	t.Run("abstract complex type rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="baseType" abstract="true">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="baseType"/>
</xs:schema>`

		instanceXML := `<root><value>hello</value></root>`

		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(t.Context(), schemaDOC)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)

		err = xsd.Validate(t.Context(), doc, schema)
		require.Error(t, err)
		require.Contains(t, err.Error(), "The type definition is abstract.")
	})

	t.Run("concrete derived type via xsi:type accepted", func(t *testing.T) {
		t.Parallel()
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

		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(t.Context(), schemaDOC)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)

		err = xsd.Validate(t.Context(), doc, schema)
		require.NoError(t, err)
	})

	t.Run("unrelated xsi:type rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="TypeA">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="TypeB">
    <xs:sequence>
      <xs:element name="b" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="TypeA"/>
</xs:schema>`

		instanceXML := `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:type="TypeB"><b>hello</b></root>`

		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(t.Context(), schemaDOC)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)

		err = xsd.Validate(t.Context(), doc, schema)
		require.Error(t, err)
		require.Contains(t, err.Error(), "is not validly derived from")
	})

	t.Run("non-existent xsi:type rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="TypeA">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="TypeA"/>
</xs:schema>`

		instanceXML := `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:type="NoSuchType"><a>hello</a></root>`

		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(t.Context(), schemaDOC)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)

		err = xsd.Validate(t.Context(), doc, schema)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not resolve to a type definition")
	})

	t.Run("same xsi:type as declared accepted", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="TypeA">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="TypeA"/>
</xs:schema>`

		instanceXML := `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:type="TypeA"><a>hello</a></root>`

		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(t.Context(), schemaDOC)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)

		err = xsd.Validate(t.Context(), doc, schema)
		require.NoError(t, err)
	})

	t.Run("restriction xsi:type accepted", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="BaseType">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="RestrictedType">
    <xs:complexContent>
      <xs:restriction base="BaseType">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="BaseType"/>
</xs:schema>`

		instanceXML := `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:type="RestrictedType"><a>hello</a></root>`

		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(t.Context(), schemaDOC)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)

		err = xsd.Validate(t.Context(), doc, schema)
		require.NoError(t, err)
	})

	t.Run("non-abstract type accepted", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="myType">
    <xs:sequence>
      <xs:element name="value" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="root" type="myType"/>
</xs:schema>`

		instanceXML := `<root><value>hello</value></root>`

		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)

		schema, err := xsd.Compile(t.Context(), schemaDOC)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
		require.NoError(t, err)

		err = xsd.Validate(t.Context(), doc, schema)
		require.NoError(t, err)
	})
}

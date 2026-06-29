package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11LocalAttributeTargetNamespace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="TEST_TYPE">
          <xs:attribute name="a" type="xs:integer" targetNamespace="http://test1"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="TEST_TYPE">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:anyAttribute/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<x xmlns:test1="http://test1" test1:a="100">Hello World</x>`))
	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<x a="100">Hello World</x>`), xsd.ErrValidationFailed)
}

func TestVersion11LocalAttributeTargetNamespaceRepresentationConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "requires name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="x">
    <xs:complexType>
      <xs:attribute targetNamespace="urn:other" type="xs:string"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "not allowed with ref",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="x">
    <xs:complexType>
      <xs:attribute ref="a" targetNamespace="urn:other"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "not allowed with form",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:anyAttribute/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:attribute name="a" type="xs:string" targetNamespace="urn:other" form="qualified"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed in extension",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="base">
          <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed in attribute group",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="attrs">
    <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
  </xs:attributeGroup>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed in anyType restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="xs:anyType">
        <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "cross namespace not allowed on nested element attribute inside restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="item" type="xs:anyType"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="restricted">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="item">
            <xs:complexType>
              <xs:attribute name="a" type="xs:string" targetNamespace="urn:other"/>
            </xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "xsi namespace not allowed in valid restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:anyAttribute/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:attribute name="local" type="xs:string" targetNamespace="http://www.w3.org/2001/XMLSchema-instance"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)

			_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}

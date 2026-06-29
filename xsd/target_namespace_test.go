package xsd_test

import (
	"testing"

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

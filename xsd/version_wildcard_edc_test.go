package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11WildcardMatchedElementConsistent(t *testing.T) {
	t.Parallel()

	t.Run("rejects incompatible wildcard governing type", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="b"
  xmlns:b="b"
  elementFormDefault="qualified">
  <xs:complexType name="t">
    <xs:sequence>
      <xs:element name="x" type="xs:string" minOccurs="0"/>
      <xs:any namespace="b" processContents="lax" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="x" type="xs:integer"/>
  <xs:element name="root" type="b:t"/>
</xs:schema>`

		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<root xmlns="b"><x>a</x><x>3</x></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("accepts derived wildcard governing type", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="b"
  xmlns:b="b"
  elementFormDefault="qualified">
  <xs:complexType name="t">
    <xs:sequence>
      <xs:element name="x" type="xs:decimal" minOccurs="0"/>
      <xs:any namespace="b" processContents="lax" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="x" type="xs:integer"/>
  <xs:element name="root" type="b:t"/>
</xs:schema>`

		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<root xmlns="b"><x>1.5</x><x>3</x></root>`)
		require.NoError(t, err)
	})
}

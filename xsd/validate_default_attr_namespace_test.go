package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestValidateDefaultAttributeNamespaceShadowing(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:p="urn:target" xmlns:q="urn:target" targetNamespace="urn:target"
    elementFormDefault="qualified">
  <xs:attribute name="a" type="xs:string" default="v"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence><xs:element ref="q:child"/></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="child">
    <xs:complexType><xs:attribute ref="p:a"/></xs:complexType>
  </xs:element>
</xs:schema>`

	schema, errs := compileWithErrors(t, schemaXML)
	require.Empty(t, errs)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<q:root xmlns:p="urn:target" xmlns:q="urn:target"><q:child xmlns:p="urn:other"/></q:root>`,
	))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), doc))

	serialized, err := helium.WriteString(doc)
	require.NoError(t, err)
	roundTrip, err := helium.NewParser().Parse(t.Context(), []byte(serialized))
	require.NoError(t, err)
	child := roundTrip.DocumentElement().FirstChild().(*helium.Element)
	value, found := child.GetAttributeNS("a", "urn:target")
	require.True(t, found, "serialized default attribute must retain its declared namespace: %s", serialized)
	require.Equal(t, "v", value)
}

package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestXsiTypeCanGovernUndeclaredRoot(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="RootType">
    <xs:sequence>
      <xs:element name="n" type="xs:int"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML,
		`<root `+xsiNS+` xsi:type="RootType"><n>7</n></root>`))
	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML,
		`<root `+xsiNS+` xsi:type="RootType"><n>not-int</n></root>`), xsd.ErrValidationFailed)
	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML,
		`<root><n>7</n></root>`), xsd.ErrValidationFailed)
}

func TestXsiTypeUnboundPrefixDoesNotResolveToNoNamespaceType(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="RootType">
    <xs:sequence>
      <xs:element name="n" type="xs:int"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	const instanceXML = `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="missing:RootType"><n>7</n></root>`

	err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML, instanceXML)
	require.ErrorIs(t, err, xsd.ErrValidationFailed)
}

func TestXsiTypeDoesNotFallbackToSchemaTargetNamespace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="base">
  </xs:complexType>
  <xs:complexType name="ext">
    <xs:complexContent>
      <xs:extension base="t:base"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:base"/>
</xs:schema>`

	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML,
		`<t:root xmlns:t="urn:t" xmlns="urn:wrong" `+xsiNS+` xsi:type="t:ext"/>`))
	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML,
		`<t:root xmlns:t="urn:t" xmlns="urn:wrong" `+xsiNS+` xsi:type="ext"/>`),
		xsd.ErrValidationFailed)
}

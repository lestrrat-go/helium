package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11XSITypeUnionMember(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="http://xstest-tns/IBMd3_16v05"
    xmlns:sv="http://xstest-tns/IBMd3_16v05">
  <xs:simpleType name="u_string">
    <xs:restriction base="xs:string">
      <xs:pattern value="[1-9][1-9]"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u_integer">
    <xs:restriction base="xs:integer">
      <xs:pattern value="[3-4][3-4]"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u_ncname">
    <xs:restriction base="xs:NCName">
      <xs:pattern value="[a-z][x-z]"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="unionAll">
    <xs:union memberTypes="sv:u_ncname sv:u_integer sv:u_string"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="union_element" type="sv:unionAll" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	const instanceXML = `<dv:root xmlns:dv="http://xstest-tns/IBMd3_16v05"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <union_element xsi:type="dv:u_string">28</union_element>
  <union_element xsi:type="dv:u_integer">33</union_element>
  <union_element xsi:type="dv:u_ncname">az</union_element>
</dv:root>`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML, instanceXML))
}

func TestVersion11XSITypeUnionMemberGuards(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="SmallInt">
    <xs:restriction base="xs:integer">
      <xs:maxInclusive value="100"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="SmallOrBool">
    <xs:union memberTypes="SmallInt xs:boolean"/>
  </xs:simpleType>
  <xs:element name="open" type="SmallOrBool"/>
  <xs:element name="blocked" type="SmallOrBool" block="restriction"/>
</xs:schema>`

	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	t.Run("unrelated xsi:type is still rejected", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<open `+xsiNS+` xsi:type="xs:string">value</open>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("block restriction still rejects union member xsi:type", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
			`<blocked `+xsiNS+` xsi:type="SmallInt">5</blocked>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})
}

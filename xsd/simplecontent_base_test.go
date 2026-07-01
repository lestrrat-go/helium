package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSimpleContentBase locks src-ct.2 (Complex Type Definition Representation
// OK, XSD Structures §3.4.2): a <xs:simpleContent> derivation's @base must be of
// the right KIND and content. The rule is version-INDEPENDENT, so the DEFAULT
// (1.0) compiler must reject the invalid forms while still accepting the valid
// canonical ones. Invalid cases mirror W3C msData/complexType ctD001/ctD004/
// ctE003/ctE004/ctK002/ctM001, which helium's 1.0 mode used to accept.
func TestSimpleContentBase(t *testing.T) {
	t.Parallel()

	wrap := func(body string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xs:schema>`
	}

	invalid := map[string]string{
		// ctD001: restriction whose base is a simple type (clause 2.2 is extension-only).
		"restriction base is builtin simple type": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:restriction base="xs:string"/></xs:simpleContent>
  </xs:complexType>`),
		// ctM001: restriction whose base is a user simple type.
		"restriction base is user simple type": wrap(`
  <xs:simpleType name="myType"><xs:restriction base="xs:string"/></xs:simpleType>
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:restriction base="myType"><xs:length value="5"/></xs:restriction></xs:simpleContent>
  </xs:complexType>`),
		// ctD004: restriction whose base is xs:anyType (mixed+emptiable) but the
		// restriction carries no nested <xs:simpleType> (clause 2.3 requires one).
		"restriction base is anyType without simpleType child": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:restriction base="xs:anyType"/></xs:simpleContent>
  </xs:complexType>`),
		// ctE003: extension whose base is a complex type with complex content.
		"extension base has complex content": wrap(`
  <xs:complexType name="myType">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:extension base="myType"/></xs:simpleContent>
  </xs:complexType>`),
		// ctE004: extension whose base is xs:anyType (mixed complex content).
		"extension base is anyType": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:extension base="xs:anyType"/></xs:simpleContent>
  </xs:complexType>`),
		// ctK002: extension whose base is a mixed complex type (invalid at the
		// intermediate type, so the whole schema is invalid).
		"extension base is mixed complex type": wrap(`
  <xs:complexType name="myCT" mixed="true">
    <xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="myType">
    <xs:simpleContent><xs:extension base="myCT"/></xs:simpleContent>
  </xs:complexType>`),
	}

	valid := map[string]string{
		"extension base is builtin simple type": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="a" type="xs:string"/></xs:extension></xs:simpleContent>
  </xs:complexType>`),
		"extension base is complex simple-content type": wrap(`
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:extension base="base"><xs:attribute name="a" type="xs:string"/></xs:extension></xs:simpleContent>
  </xs:complexType>`),
		"restriction base is complex simple-content type": wrap(`
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:restriction base="base"><xs:maxLength value="5"/></xs:restriction></xs:simpleContent>
  </xs:complexType>`),
		// Clause 2.3: restriction of a mixed+emptiable complex base WITH a nested
		// <xs:simpleType> is valid.
		"restriction of mixed emptiable base with simpleType child": wrap(`
  <xs:complexType name="base" mixed="true">
    <xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="fooType">
    <xs:simpleContent>
      <xs:restriction base="base">
        <xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="5"/></xs:restriction></xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>`),
	}

	for name, schema := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			_, err := compileV10(t, schema)
			require.Error(t, err, "XSD 1.0 must reject: %s", name)
		})
	}
	for name, schema := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			_, err := compileV10(t, schema)
			require.NoError(t, err, "XSD 1.0 must accept: %s", name)
		})
	}
}

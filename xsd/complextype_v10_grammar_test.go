package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestComplexTypeV10Grammar locks the XSD 1.0 enforcement of the complexType
// schema-representation grammar rules that were previously only enforced in 1.1
// (or not at all). Each grammar/derivation-body rule below is version-INDEPENDENT
// per XSD 1.0 Structures §3.4.2, so the DEFAULT (1.0) compiler must reject the
// invalid form while still accepting the valid canonical forms. The invalid cases
// mirror W3C msData/complexType ctB*/ctC*/ctD*/ctE*/ctF*/ctG*/ctH*/ctZ*/ctA* that
// helium's 1.0 mode used to accept.
func TestComplexTypeV10Grammar(t *testing.T) {
	t.Parallel()

	wrap := func(body string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xs:schema>`
	}

	invalid := map[string]string{
		// Direct complexType grammar: (annotation?, ...) — annotation after content.
		"content model then annotation": wrap(`
  <xs:complexType name="fooType">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:annotation><xs:documentation>late</xs:documentation></xs:annotation>
  </xs:complexType>`),
		"attribute then annotation": wrap(`
  <xs:complexType name="fooType">
    <xs:attribute name="a"/>
    <xs:annotation><xs:documentation>late</xs:documentation></xs:annotation>
  </xs:complexType>`),
		"two direct annotations": wrap(`
  <xs:complexType name="fooType">
    <xs:annotation><xs:documentation>one</xs:documentation></xs:annotation>
    <xs:annotation><xs:documentation>two</xs:documentation></xs:annotation>
  </xs:complexType>`),

		// complexContent/simpleContent wrapper grammar: (annotation?, (restriction|extension)).
		"simpleContent with no derivation": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent>
      <xs:annotation><xs:documentation>only annotation</xs:documentation></xs:annotation>
    </xs:simpleContent>
  </xs:complexType>`),
		"complexContent with no content": wrap(`
  <xs:complexType name="fooType"><xs:complexContent></xs:complexContent></xs:complexType>`),
		"simpleContent extension then annotation": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
      <xs:annotation><xs:documentation>late</xs:documentation></xs:annotation>
    </xs:simpleContent>
  </xs:complexType>`),
		"complexContent restriction then annotation": wrap(`
  <xs:complexType name="base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="fooType">
    <xs:complexContent>
      <xs:restriction base="base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>
      <xs:annotation><xs:documentation>late</xs:documentation></xs:annotation>
    </xs:complexContent>
  </xs:complexType>`),

		// simpleContent restriction body grammar.
		"simpleContent restriction with model group": wrap(`
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
  <xs:complexType name="fooType">
    <xs:simpleContent>
      <xs:restriction base="base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>
    </xs:simpleContent>
  </xs:complexType>`),
		"simpleContent restriction attribute then facet": wrap(`
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
  <xs:complexType name="fooType">
    <xs:simpleContent>
      <xs:restriction base="base"><xs:attribute name="a"/><xs:length value="5"/></xs:restriction>
    </xs:simpleContent>
  </xs:complexType>`),

		// simpleContent extension body grammar: no facets, no model group.
		"simpleContent extension with facet": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:extension base="xs:string"><xs:length value="5"/></xs:extension></xs:simpleContent>
  </xs:complexType>`),
		"simpleContent extension with model group": wrap(`
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:extension base="xs:string"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:extension></xs:simpleContent>
  </xs:complexType>`),

		// complexContent derivation-body grammar: content then annotation.
		"complexContent restriction content then annotation": wrap(`
  <xs:complexType name="base"><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType>
  <xs:complexType name="fooType">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
        <xs:annotation><xs:documentation>late</xs:documentation></xs:annotation>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>`),

		// §3.4.6.2 extension mixed-agreement (cos-ct-extends): base element-only,
		// derived mixed.
		"extension mixes an element-only base": wrap(`
  <xs:complexType name="base"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice></xs:complexType>
  <xs:complexType name="fooType">
    <xs:complexContent mixed="true"><xs:extension base="base"/></xs:complexContent>
  </xs:complexType>`),

		// complexType @name must be a valid NCName.
		"complexType name with colon": wrap(`
  <xs:complexType name="a:b"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`),
		"complexType name starting with a digit": wrap(`
  <xs:complexType name="1foo"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`),
	}

	valid := map[string]string{
		"annotation then simpleContent extension": wrap(`
  <xs:complexType name="fooType">
    <xs:annotation><xs:documentation>doc</xs:documentation></xs:annotation>
    <xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="a" type="xs:string"/></xs:extension></xs:simpleContent>
  </xs:complexType>`),
		"simpleContent restriction with facet then attribute": wrap(`
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="a" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="fooType">
    <xs:simpleContent><xs:restriction base="base"><xs:maxLength value="5"/><xs:attribute name="a" type="xs:string"/></xs:restriction></xs:simpleContent>
  </xs:complexType>`),
		"complexContent restriction with model group then attributes": wrap(`
  <xs:complexType name="base"><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence><xs:attribute name="x"/></xs:complexType>
  <xs:complexType name="fooType">
    <xs:complexContent><xs:restriction base="base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="x"/></xs:restriction></xs:complexContent>
  </xs:complexType>`),
		"complexContent extension preserves mixed": wrap(`
  <xs:complexType name="base" mixed="true"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice></xs:complexType>
  <xs:complexType name="fooType">
    <xs:complexContent mixed="true"><xs:extension base="base"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent>
  </xs:complexType>`),
		"mixed complexType with model group and attributes": wrap(`
  <xs:complexType name="fooType" mixed="true">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
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

package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRestrictionAttrInheritance10 covers XSD 1.0 §3.4.2 complex type
// {attribute uses}: a complex type derived by RESTRICTION inherits each base
// attribute use (and the base {attribute wildcard}) it does not redeclare or
// prohibit. An instance carrying such an inherited attribute must validate.
// These mirror the W3C msData ctF013/ctG001/ctG013, particlesZ002 and sunData
// combined/009 conformance cases (all expected VALID).
func TestRestrictionAttrInheritance10(t *testing.T) {
	t.Parallel()

	t.Run("complexContent restriction inherits base attribute", func(t *testing.T) {
		t.Parallel()
		// ctF013: fooType restricts myType (which declares myAttr); the instance
		// carries the inherited myAttr.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="myType">
    <xs:choice>
      <xs:element name="myElement" type="xs:string"/>
      <xs:element name="myElement2" type="xs:string"/>
    </xs:choice>
    <xs:attribute name="myAttr"/>
  </xs:complexType>
  <xs:complexType name="fooType">
    <xs:complexContent>
      <xs:restriction base="myType">
        <xs:sequence>
          <xs:element name="myElement" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="fooType"/>
</xs:schema>`
		instanceXML := `<root myAttr="test attribute"><myElement>test data</myElement></root>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("restriction with no explicit override inherits all base attributes", func(t *testing.T) {
		t.Parallel()
		// sunData combined/009 test.3.v: 'default' restricts 'base' with an empty
		// restriction; a,b,c are all inherited.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="urn:foo" targetNamespace="urn:foo" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:attribute name="a" type="xs:string"/>
    <xs:attribute name="b" type="xs:string"/>
    <xs:attribute name="c" type="xs:string"/>
  </xs:complexType>
  <xs:element name="default">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="base"/>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<foo:default xmlns:foo="urn:foo" a="xxx" b="xxx" c="xxx"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("restriction that prohibits one attribute inherits the rest", func(t *testing.T) {
		t.Parallel()
		// sunData combined/009 test.11.v: 'prohibit' restricts 'base' prohibiting c;
		// a,b inherited, c absent from the instance.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="urn:foo" targetNamespace="urn:foo" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:attribute name="a" type="xs:string"/>
    <xs:attribute name="b" type="xs:string"/>
    <xs:attribute name="c" type="xs:string"/>
  </xs:complexType>
  <xs:element name="prohibit">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="base">
          <xs:attribute name="c" use="prohibited"/>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<foo:prohibit xmlns:foo="urn:foo" a="xxx" b="xxx"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("restriction over a wildcard base inherits the named attribute use", func(t *testing.T) {
		t.Parallel()
		// particlesZ002 Derived3: restricts Base3 (foo + anyAttribute ##local),
		// prohibiting bar; foo is inherited as an attribute USE (not via the
		// wildcard, which a restriction does not inherit), so foo on the instance
		// matches the inherited use.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base3">
    <xs:attribute name="foo"/>
    <xs:anyAttribute namespace="##local"/>
  </xs:complexType>
  <xs:complexType name="Derived3">
    <xs:complexContent>
      <xs:restriction base="Base3">
        <xs:attribute name="bar" use="prohibited"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived3"/>
</xs:schema>`
		instanceXML := `<root foo="123"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("empty restriction does NOT inherit the base attribute wildcard", func(t *testing.T) {
		t.Parallel()
		// sunData combined/008 alias/test.10.n: 'alias' is an empty restriction of a
		// base carrying anyAttribute; in XSD 1.0 the restriction's wildcard is
		// computed solely from its own content, so it has NONE — an instance
		// attribute the base wildcard would admit is rejected.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a" targetNamespace="urn:foo" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:anyAttribute namespace="urn:a urn:b" processContents="skip"/>
  </xs:complexType>
  <xs:element name="alias">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="base"/>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<foo:alias xmlns:foo="urn:foo" xmlns:a="urn:a" a:xxx="xxx"/>`
		require.Error(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})

	t.Run("simpleContent restriction inherits base attribute", func(t *testing.T) {
		t.Parallel()
		// ctE019: fooType simpleContent-restricts mytype1 (attrTest1, attrTest2),
		// redeclaring only attrTest1; attrTest2 is inherited.
		schemaXML := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns="a">
  <xsd:simpleType name="myType">
    <xsd:restriction base="xsd:string"/>
  </xsd:simpleType>
  <xsd:complexType name="mytype1">
    <xsd:simpleContent>
      <xsd:extension base="myType">
        <xsd:attribute ref="attrTest1"/>
        <xsd:attribute ref="attrTest2"/>
      </xsd:extension>
    </xsd:simpleContent>
  </xsd:complexType>
  <xsd:complexType name="fooType">
    <xsd:simpleContent>
      <xsd:restriction base="mytype1">
        <xsd:attribute ref="attrTest1"/>
      </xsd:restriction>
    </xsd:simpleContent>
  </xsd:complexType>
  <xsd:attribute name="attrTest1" type="xsd:int"/>
  <xsd:attribute name="attrTest2" type="xsd:ID"/>
  <xsd:element name="doc" type="fooType"/>
</xsd:schema>`
		instanceXML := `<a:doc xmlns:a="a" a:attrTest1="1" a:attrTest2="b"/>`
		require.NoError(t, compileAndValidate(t, schemaXML, instanceXML, nil))
	})
}

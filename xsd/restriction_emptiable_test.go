package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// addB118: a complexContent restriction may narrow an emptiable base choice's
// minOccurs below the base's declared minOccurs in XSD 1.1 (§3.9.6 treats an
// emptiable base particle as having effective minOccurs 0), but XSD 1.0 keeps
// the strict rMin >= bMin rule. Here the base outer choice is emptiable via its
// minOccurs="0" sequence branch, and the derived choice narrows minOccurs from
// the default 1 down to 0 — valid in 1.1, invalid in 1.0.
func TestRestriction_EmptiableBaseChoiceMinOccurs(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="restrictionType">
    <xs:sequence>
      <xs:element name="f"/>
      <xs:choice>
        <xs:choice minOccurs="1">
          <xs:element name="a"/>
          <xs:element name="b"/>
        </xs:choice>
        <xs:sequence minOccurs="0">
          <xs:element name="a1"/>
          <xs:element name="b1"/>
        </xs:sequence>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="complexRestrictionType">
    <xs:complexContent>
      <xs:restriction base="restrictionType">
        <xs:sequence>
          <xs:element name="f"/>
          <xs:choice minOccurs="0">
            <xs:element name="a"/>
            <xs:element name="b"/>
          </xs:choice>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	// XSD 1.1: the base outer choice is emptiable (its sequence branch has
	// minOccurs="0"), so narrowing the derived choice to minOccurs="0" is a
	// valid restriction.
	schema11, _, cerr := compileV11(t, schemaXML)
	require.NoError(t, cerr, "1.1: an emptiable base choice may be narrowed to minOccurs=0")
	require.NotNil(t, schema11)

	// XSD 1.0: the strict rMin >= bMin rule rejects the same restriction.
	_, cerr10 := compileV10(t, schemaXML)
	require.ErrorIs(t, cerr10, xsd.ErrCompilationFailed, "1.0: derived minOccurs 0 below base minOccurs 1 is invalid")
}

// The same emptiable-base occurrence relaxation applies to an xs:all restricting
// an xs:all: a base all{1,1} whose members are all optional is emptiable, so a
// derived all{0,1} narrowing the group's minOccurs below the base's declared 1
// is a valid restriction in XSD 1.1 (§3.9.6) but rejected under XSD 1.0's strict
// rMin >= bMin rule. A base whose member is REQUIRED is NOT emptiable, so the
// same narrowing stays invalid in both versions.
func TestRestriction_EmptiableBaseAllMinOccurs(t *testing.T) {
	t.Parallel()

	// Base all is emptiable (both members optional) → derived all{0,1} is valid @1.1.
	const validXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:s" xmlns:s="urn:s" elementFormDefault="qualified">
  <xs:complexType name="Base">
    <xs:all minOccurs="1" maxOccurs="1">
      <xs:element name="a" type="xs:string" minOccurs="0"/>
      <xs:element name="b" type="xs:string" minOccurs="0"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="s:Base">
        <xs:all minOccurs="0" maxOccurs="1">
          <xs:element name="a" type="xs:string" minOccurs="0"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="s:Derived"/>
</xs:schema>`

	schema11, _, cerr := compileV11(t, validXML)
	require.NoError(t, cerr, "1.1: an emptiable base all may be narrowed to minOccurs=0")
	require.NotNil(t, schema11)

	_, cerr10 := compileV10(t, validXML)
	require.ErrorIs(t, cerr10, xsd.ErrCompilationFailed, "1.0: derived all minOccurs 0 below base minOccurs 1 is invalid")

	// Base all is NOT emptiable (member a is required) → the same narrowing is
	// invalid in 1.1 too (the base does not accept the empty sequence).
	const invalidXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:s" xmlns:s="urn:s" elementFormDefault="qualified">
  <xs:complexType name="Base">
    <xs:all minOccurs="1" maxOccurs="1">
      <xs:element name="a" type="xs:string" minOccurs="1"/>
      <xs:element name="b" type="xs:string" minOccurs="0"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="s:Base">
        <xs:all minOccurs="0" maxOccurs="1">
          <xs:element name="a" type="xs:string" minOccurs="0"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="s:Derived"/>
</xs:schema>`

	_, _, cerrInvalid := compileV11(t, invalidXML)
	require.ErrorIs(t, cerrInvalid, xsd.ErrCompilationFailed, "1.1: a non-emptiable base all may not be narrowed below its declared minOccurs")
}

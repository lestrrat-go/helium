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

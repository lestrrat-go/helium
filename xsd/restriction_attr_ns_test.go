package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRestrictionAttrNamespace (XSD-003) verifies that the
// derivation-ok-restriction attribute checks key base/derived attribute uses on
// the full QName (namespace + local), not just the local name. A base type that
// requires a namespaced attribute {urn:t}id must NOT be considered restricted by
// a derived restriction that declares an unqualified attribute named "id":
// the two share a local name but live in different namespaces.
func TestRestrictionAttrNamespace(t *testing.T) {
	t.Parallel()

	const noMatchingUse = "Neither a matching attribute use, nor a matching wildcard exists"
	const requiredMissing = "A matching attribute use for the 'required' attribute use"

	t.Run("rejects unqualified derived attr restricting namespaced base attr", func(t *testing.T) {
		t.Parallel()
		// Base requires the namespaced global attribute {urn:t}id; the derived
		// restriction declares a local unqualified attribute "id" ({}id). They
		// collide only on local name, so the derivation must be rejected: the
		// required base {urn:t}id has no counterpart and the derived {}id matches
		// neither a base attribute use nor a base wildcard.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" attributeFormDefault="unqualified">
  <xs:attribute name="id" type="xs:string"/>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute ref="t:id" use="required"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="id" type="xs:string" use="required"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		errs := compileFatalErrors(t, schema)
		require.Contains(t, errs, noMatchingUse)
		require.Contains(t, errs, requiredMissing)
	})

	t.Run("accepts namespaced derived attr restricting same namespaced base attr", func(t *testing.T) {
		t.Parallel()
		// Control: derived restriction reuses the same namespaced {urn:t}id; the
		// QNames match, so this is a valid (identity) restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" attributeFormDefault="unqualified">
  <xs:attribute name="id" type="xs:string"/>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute ref="t:id" use="required"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute ref="t:id" use="required"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}

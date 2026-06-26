package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAttrWildcardRestrictionSubset (XSD-004) verifies that the
// derivation-ok-restriction §3.10.6 attribute-wildcard subset check is
// negation-aware. The earlier helper folded the negation constraints
// (##other/##not-absent) into empty namespace sets, which both false-accepted a
// derived ##other against a finite base set and false-rejected a finite derived
// set against a base ##other.
func TestAttrWildcardRestrictionSubset(t *testing.T) {
	t.Parallel()

	const notValidSubset = "is not a valid subset of the wildcard"

	t.Run("rejects derived ##other against base ##local", func(t *testing.T) {
		t.Parallel()
		// Base attribute wildcard admits only the absent namespace (##local).
		// Derived ##other admits infinitely many namespaces — NOT a subset, so
		// the restriction must be rejected. The buggy helper folded both into
		// empty sets and wrongly accepted it.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:anyAttribute namespace="##local" processContents="lax"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:anyAttribute namespace="##other" processContents="lax"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidSubset)
	})

	t.Run("accepts finite derived set against base ##other", func(t *testing.T) {
		t.Parallel()
		// Base attribute wildcard is ##other (admits every namespace except the
		// target namespace urn:t and the absent namespace). Derived restricts to
		// the finite namespace urn:o, which the base admits — a valid subset. The
		// buggy helper folded ##other into the empty set and wrongly rejected it.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:anyAttribute namespace="##other" processContents="lax"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:anyAttribute namespace="urn:o" processContents="lax"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}

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

	t.Run("rejects derived attr outside base wildcard namespace", func(t *testing.T) {
		t.Parallel()
		// Base has an attribute wildcard restricted to ##targetNamespace
		// (urn:t). The derived restriction declares an unqualified ({}foo)
		// attribute, whose absent namespace is NOT admitted by the base
		// wildcard, so the derivation must be rejected: no matching base
		// attribute use and no base wildcard that covers {}foo.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" attributeFormDefault="unqualified">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:anyAttribute namespace="##targetNamespace"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="foo" type="xs:string"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), noMatchingUse)
	})

	t.Run("accepts restriction adding wildcard when base wildcard is transitive via attribute group", func(t *testing.T) {
		t.Parallel()
		// The base type declares NO direct xs:anyAttribute; its attribute wildcard
		// comes transitively through a referenced attribute group (g -> nested g2
		// with <xs:anyAttribute namespace="##other">). A restriction that adds its
		// own ##other wildcard is therefore a valid subset of the base's effective
		// (complete) attribute wildcard and must NOT be rejected with "the base
		// complex type definition does not have one". This is the schema-for-xslt20
		// 'transform-element-base-type' shape; it must compile in XSD 1.0 (default).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attributeGroup name="g2">
    <xs:attribute name="a" type="xs:string"/>
    <xs:anyAttribute namespace="##other" processContents="lax"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g">
    <xs:attributeGroup ref="t:g2"/>
    <xs:attribute name="b" type="xs:string"/>
  </xs:attributeGroup>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attributeGroup ref="t:g"/>
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
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("narrows a direct base wildcard by a group-ref wildcard", func(t *testing.T) {
		t.Parallel()
		// The base has a DIRECT <xs:anyAttribute namespace="##any"/> AND a
		// referenced attribute group contributing a NARROWER ##other wildcard.
		// The effective complete wildcard is their intersection, so a derived
		// ##any wildcard widens the base and is not a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
    <xs:anyAttribute namespace="##other" processContents="lax"/>
  </xs:attributeGroup>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attributeGroup ref="t:g"/>
    <xs:anyAttribute namespace="##any"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:anyAttribute namespace="##any"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "not a valid subset")
	})

	t.Run("rejects restriction widening a transitive base attribute-group wildcard", func(t *testing.T) {
		t.Parallel()
		// Control: the base's effective wildcard (via the attribute group) is
		// restricted to ##other; a derived ##any wildcard is a WIDENING, not a
		// subset, so the derivation must still be rejected. This proves the fix
		// compares against the real transitive wildcard rather than blanket-accepting.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
    <xs:anyAttribute namespace="##other" processContents="lax"/>
  </xs:attributeGroup>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attributeGroup ref="t:g"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:anyAttribute namespace="##any" processContents="lax"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), "not a valid subset")
	})

	t.Run("accepts derived attr admitted by base wildcard namespace", func(t *testing.T) {
		t.Parallel()
		// Base has an attribute wildcard covering ##targetNamespace (urn:t).
		// The derived restriction declares a {urn:t}foo attribute (a global,
		// hence target-namespace, attribute), which the base wildcard admits,
		// so the derivation is valid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" attributeFormDefault="qualified">
  <xs:attribute name="foo" type="xs:string"/>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:anyAttribute namespace="##targetNamespace"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute ref="t:foo"/>
        <xs:anyAttribute namespace="##targetNamespace"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}

package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_InterleaveOpenChildEDC covers the gauntlet finding that an
// interleave-refined child moved into the open sub-sequence must still get the
// SAME dynamic wildcard Element-Declarations-Consistent (EDC) check as the ordinary
// wildcard-particle path: a child whose name collides with a same-named local
// element whose type is inconsistent with the wildcard's governing (global) type
// must be rejected.
func TestOpenContent_InterleaveOpenChildEDC(t *testing.T) {
	t.Parallel()

	t.Run("inconsistent local int vs global duration is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:duration"/>
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		// First <e> is the local int (valid); the second is open content matched by
		// the lax ##local wildcard, lax-resolving to global e=duration. The EDC check
		// must reject it (local int vs governing duration inconsistent).
		require.Error(t, validateOC(t, schema, `<doc><e>1</e><e>PT1H</e></doc>`),
			"open-content child colliding with an inconsistent local declaration must be rejected")
	})

	t.Run("consistent local int and global int is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc><e>1</e><e>2</e></doc>`),
			"a consistent local/global type must not be rejected by the EDC guard")
	})
}

// TestOpenContent_RestrictionDropsBaseLocal covers the gauntlet finding that a
// restriction must not DROP a base local element declaration and re-admit the same
// name through a lenient open-content wildcard (skip, or lax with no global
// declaration) that does not enforce the base's declared type.
func TestOpenContent_RestrictionDropsBaseLocal(t *testing.T) {
	t.Parallel()

	t.Run("dropped base local re-admitted by a skip wildcard is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a base local element re-admitted by a skip wildcard must be rejected")
	})

	t.Run("dropped base local re-admitted by lax-with-no-global is rejected", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "dropping a base local element re-admitted by lax-with-no-global must be rejected")
	})

	t.Run("keeping the base local element is accepted", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a restriction that KEEPS the base local element is valid")
	})

	t.Run("dropped base local re-admitted by strict wildcard with consistent global is accepted", func(t *testing.T) {
		t.Parallel()
		// A strict wildcard resolves the name to the global declaration and the
		// dynamic EDC enforces type consistency at validation, so dropping the local
		// is a valid restriction (not a compile-time error).
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:int"/>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="strict"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a strict wildcard re-admitting the name (dynamic EDC enforces) must not be rejected at compile")
	})
}

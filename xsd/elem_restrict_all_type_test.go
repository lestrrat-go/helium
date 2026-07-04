package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestElementRestrictAll_TypeGate covers the NameAndTypeOK type-derivation gate on
// the XSD 1.1 xs:all occurrence-counting restriction path (all_subsumption.go
// derivedElemNameAndTypeOK), which shares elementTypeValidlyRestricts with the
// direct element:element path (elementRestrictsElement). An xs:all element retyped
// to an EXTENSION-derived type is rejected (clause 3.2.5.2), a retyping the base
// TYPE's @block forbids is rejected (cvc-elt.4.3), while a valid RESTRICTION
// retyping — and a list-from-xs:anySimpleType retyping — stays accepted. xs:all
// occurrence-counting is 1.1-only, so the whole gate exercises Version11.
func TestElementRestrictAll_TypeGate(t *testing.T) {
	// ContBase carries an xs:all with element x typed XBase; ContDer restricts it
	// with x retyped to xType. block is applied to XBase's {prohibited substitutions}.
	schemaFor := func(block, xType string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="XBase" ` + block + `>
    <xs:sequence><xs:element name="p" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="XExt">
    <xs:complexContent>
      <xs:extension base="t:XBase">
        <xs:sequence><xs:element name="q" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="XRes">
    <xs:complexContent>
      <xs:restriction base="t:XBase">
        <xs:sequence><xs:element name="p" type="xs:string"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="ContBase">
    <xs:all><xs:element name="x" type="t:XBase"/></xs:all>
  </xs:complexType>
  <xs:complexType name="ContDer">
    <xs:complexContent>
      <xs:restriction base="t:ContBase">
        <xs:all><xs:element name="x" type="` + xType + `"/></xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="t:ContDer"/>
</xs:schema>`
	}

	t.Run("extension retyping is rejected", func(t *testing.T) {
		_, _, cerr := compileV11(t, schemaFor("", "t:XExt"))
		require.Error(t, cerr)
	})
	t.Run("base TYPE block=restriction rejects restriction retyping", func(t *testing.T) {
		_, _, cerr := compileV11(t, schemaFor(`block="restriction"`, "t:XRes"))
		require.Error(t, cerr)
	})
	t.Run("valid restriction retyping is accepted", func(t *testing.T) {
		_, _, cerr := compileV11(t, schemaFor("", "t:XRes"))
		require.NoError(t, cerr)
	})

	// A list type derived from xs:anySimpleType validly retypes a base element typed
	// xs:anySimpleType (§3.16.3 clause 2.2.3): the gate must NOT block the list arm.
	t.Run("list-from-anySimpleType retyping is accepted", func(t *testing.T) {
		const src = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="intList"><xs:list itemType="xs:int"/></xs:simpleType>
  <xs:complexType name="SBase">
    <xs:all><xs:element name="x" type="xs:anySimpleType"/></xs:all>
  </xs:complexType>
  <xs:complexType name="SDer">
    <xs:complexContent>
      <xs:restriction base="t:SBase">
        <xs:all><xs:element name="x" type="t:intList"/></xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="t:SDer"/>
</xs:schema>`
		_, _, cerr := compileV11(t, src)
		require.NoError(t, cerr)
	})
}

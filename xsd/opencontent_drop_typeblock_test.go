package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_DropsBaseLocalTypeBlock covers the soundness hole where a base
// LOCAL element's TYPE-level {prohibited substitutions} (block) — not the element
// declaration's own block — is lost when a restriction DROPS the element and
// re-admits its name through an (enforced) open-content wildcard governed by the
// GLOBAL declaration. cvc-elt.4.3 unions the element block with the declared
// type's block, so a base-local element typed with a block="extension" type
// rejects an xsi:type extension derivation the permissively-typed global admits;
// dropping+re-admitting it must therefore be rejected at compile time.
func TestOpenContent_DropsBaseLocalTypeBlock(t *testing.T) {
	t.Parallel()

	// base B declares interleave-strict open content over the target namespace and a
	// local element e typed t:Base; R restricts B keeping the same open content but
	// DROPPING e, so e spills to open content governed by the permissively-typed
	// global e. baseType is the type of the base local e.
	build := func(baseType string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="e" type="xs:anyType"/>
  <xs:complexType name="Base"` + baseType + `><xs:sequence/></xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent><xs:extension base="t:Base"><xs:sequence/></xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:element name="e" type="t:Base" minOccurs="0"/>
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent><xs:restriction base="t:B">
      <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
      <xs:sequence>
        <xs:element name="keep" type="xs:string" minOccurs="0"/>
      </xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	t.Run("dropped local whose TYPE blocks extension re-admitted via unblocked global is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(` block="extension"`)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "the base local's type blocks extension but the re-admitting global (xs:anyType) does not")
	})

	t.Run("dropped local with no type block is accepted", func(t *testing.T) {
		t.Parallel()
		schema := build(``)
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "no type block is lost, so the drop is a sound restriction")
	})
}

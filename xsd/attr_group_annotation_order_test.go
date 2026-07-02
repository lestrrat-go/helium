package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAttrGroupAnnotationOrder verifies the XML representation of an
// <xs:attributeGroup> DEFINITION content model (annotation?, ((attribute |
// attributeGroup)*, anyAttribute?)) — §3.6.2: the annotation must PRECEDE the
// attribute uses. An annotation appearing AFTER the attribute declarations is a
// schema-representation error (W3C sun AGroupDef annotation00101m3/m6). This is a
// version-INDEPENDENT rule; it is exercised under the default XSD 1.0 compiler.
func TestAttrGroupAnnotationOrder(t *testing.T) {
	t.Parallel()

	const misplaced = "The annotation must appear before the attribute declarations"

	t.Run("annotation after attributes is rejected", func(t *testing.T) {
		t.Parallel()
		// documentation-form annotation after the attribute declarations (m3).
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="G1">
    <xs:attribute name="c" type="xs:int"/>
    <xs:attribute name="date" type="xs:date"/>
    <xs:annotation>
      <xs:documentation>trailing annotation</xs:documentation>
    </xs:annotation>
  </xs:attributeGroup>
  <xs:complexType name="A">
    <xs:attributeGroup ref="G1"/>
  </xs:complexType>
  <xs:element name="root" type="A"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, misplaced,
			"an annotation after the attribute declarations must be rejected; got: %q", errs)
	})

	t.Run("annotation after an attributeGroup ref is rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="H">
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="G1">
    <xs:attributeGroup ref="H"/>
    <xs:annotation>
      <xs:documentation>trailing annotation</xs:documentation>
    </xs:annotation>
  </xs:attributeGroup>
  <xs:complexType name="A">
    <xs:attributeGroup ref="G1"/>
  </xs:complexType>
  <xs:element name="root" type="A"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, misplaced,
			"an annotation after an attributeGroup ref must be rejected; got: %q", errs)
	})

	t.Run("annotation first is accepted", func(t *testing.T) {
		t.Parallel()
		// Leading annotation (valid, mirrors sun annotation00101m1/m4).
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="G1">
    <xs:annotation>
      <xs:documentation>leading annotation</xs:documentation>
    </xs:annotation>
    <xs:attribute name="c" type="xs:int"/>
    <xs:attribute name="date" type="xs:date"/>
  </xs:attributeGroup>
  <xs:complexType name="A">
    <xs:attributeGroup ref="G1"/>
  </xs:complexType>
  <xs:element name="root" type="A"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs, "a leading annotation must be accepted; got: %q", errs)
	})
}

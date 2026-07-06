package xsd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestElementIDCTypeOrder verifies the §3.3.2 element content-model ordering
// rule that a type definition (<simpleType>/<complexType>) may not FOLLOW an
// identity constraint (<unique>/<key>/<keyref>) inside an <xs:element>: the
// content model is (annotation?, ((simpleType | complexType)?, alternative*,
// (unique | key | keyref)*)), so the type must precede every identity
// constraint. Version-INDEPENDENT. Mirrors the W3C xsd10 msMeta case idZ003
// (a <unique> declared before the element's <complexType>).
func TestElementIDCTypeOrder(t *testing.T) {
	t.Parallel()

	const wantMsg = "The content is not valid. Expected is (annotation?"

	t.Run("rejects complexType after a unique constraint", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:r="urn:r" targetNamespace="urn:r" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:unique name="u">
      <xs:selector xpath="r:regions/r:zip"/>
      <xs:field xpath="@code"/>
    </xs:unique>
    <xs:complexType>
      <xs:sequence>
        <xs:element name="regions" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		errs := compileErrorsExact(t, schemaXML)
		require.Contains(t, errs, wantMsg)
	})

	t.Run("accepts complexType before the identity constraints", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:r="urn:r" targetNamespace="urn:r" elementFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="regions" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="r:regions"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`
		errs := compileErrorsExact(t, schemaXML)
		require.False(t, strings.Contains(errs, wantMsg),
			"unexpected content-order error: %s", errs)
	})
}

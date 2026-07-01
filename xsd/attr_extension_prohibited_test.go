package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAttrExtensionProhibitedFixed verifies the XSD 1.1 Attribute Declaration
// Representation OK constraint (a prohibited attribute use must not carry a
// 'fixed' value constraint) is enforced on the xs:extension paths — both a
// complexContent extension and a simpleContent extension. These paths warn+skip
// a prohibited attribute before the ordinary attribute-use check runs, so the
// constraint is enforced at the skip site. XSD 1.0 tolerates prohibited+fixed
// everywhere, so the same schemas stay valid there.
func TestAttrExtensionProhibitedFixed(t *testing.T) {
	t.Parallel()

	const prohibitedFixed = "The attribute 'fixed' is not allowed when the value of the attribute 'use' is 'prohibited'."

	t.Run("complexContent extension prohibited+fixed", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:sequence/></xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:extension base="base">
        <xs:attribute name="a" type="xs:string" use="prohibited" fixed="x"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
		_, diag, cerr := compileV11(t, schemaXML)
		require.Error(t, cerr, "v11 must reject prohibited+fixed in a complexContent extension")
		require.Contains(t, diag, prohibitedFixed)

		_, v10err := compileV10(t, schemaXML)
		require.NoError(t, v10err, "v10 must accept prohibited+fixed in a complexContent extension")
	})

	t.Run("simpleContent extension prohibited+fixed", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="derived">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="a" type="xs:string" use="prohibited" fixed="x"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
		_, diag, cerr := compileV11(t, schemaXML)
		require.Error(t, cerr, "v11 must reject prohibited+fixed in a simpleContent extension")
		require.Contains(t, diag, prohibitedFixed)

		_, v10err := compileV10(t, schemaXML)
		require.NoError(t, v10err, "v10 must accept prohibited+fixed in a simpleContent extension")
	})

	// A prohibited attribute WITHOUT a fixed value is a legal (if pointless)
	// declaration in an extension in both versions — no over-rejection.
	t.Run("complexContent extension prohibited without fixed stays valid", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:sequence/></xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:extension base="base">
        <xs:attribute name="a" type="xs:string" use="prohibited"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
		_, _, cerr := compileV11(t, schemaXML)
		require.NoError(t, cerr, "v11 must accept prohibited (no fixed) in a complexContent extension")

		_, v10err := compileV10(t, schemaXML)
		require.NoError(t, v10err, "v10 must accept prohibited (no fixed) in a complexContent extension")
	})

	t.Run("simpleContent extension prohibited without fixed stays valid", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="derived">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="a" type="xs:string" use="prohibited"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
		_, _, cerr := compileV11(t, schemaXML)
		require.NoError(t, cerr, "v11 must accept prohibited (no fixed) in a simpleContent extension")

		_, v10err := compileV10(t, schemaXML)
		require.NoError(t, v10err, "v10 must accept prohibited (no fixed) in a simpleContent extension")
	})
}

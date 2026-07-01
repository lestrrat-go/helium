package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTABuiltinHierarchySubstitutability covers gauntlet finding
// XSDCTA-861-007: a complex simpleContent alternative whose content extends a
// built-in SUBTYPE of the declared built-in simple type is validly substitutable,
// even though the built-in type hierarchy is not BaseType-linked. Unrelated bases
// and the round-5/6 simple-vs-complex rules must still be enforced.
func TestVersion11CTABuiltinHierarchySubstitutability(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}

	// declared xs:integer + complex-simpleContent-extension-of-xs:nonNegativeInteger:
	// ACCEPTED, and the alternative governs.
	t.Run("complex simpleContent extending nonNegativeInteger is accepted and governs", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="NNIMsg">
    <xs:simpleContent>
      <xs:extension base="xs:nonNegativeInteger">
        <xs:attribute name="tag" type="xs:string" use="required"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="NNIMsg"/>
  </xs:element>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		validate := func(instance string) error {
			idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, perr)
			return xsd.NewValidator(schema).Validate(t.Context(), idoc)
		}
		// NNIMsg governs: nonNegativeInteger content + required @tag.
		require.NoError(t, validate(`<e tag="x">5</e>`))
		// -5 violates nonNegativeInteger (the declared xs:integer would have accepted it).
		require.ErrorIs(t, validate(`<e tag="x">-5</e>`), xsd.ErrValidationFailed)
		// Missing required @tag (the declared xs:integer would have accepted bare 5).
		require.ErrorIs(t, validate(`<e>5</e>`), xsd.ErrValidationFailed)
	})

	// declared xs:integer + complex-simpleContent-extension-of-xs:int: ACCEPTED
	// (xs:int ⊂ xs:long ⊂ xs:integer).
	t.Run("complex simpleContent extending int is accepted", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="IntMsg">
    <xs:simpleContent>
      <xs:extension base="xs:int">
        <xs:attribute name="tag" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="IntMsg"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})

	// declared xs:integer + complex-simpleContent-extension-of-xs:string: REJECTED
	// (xs:string is not derived from xs:integer).
	t.Run("complex simpleContent extending string is rejected", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="StrMsg">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="tag" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="StrMsg"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// Guard the round-5/6 rules: a plain simple alternative (xs:nonNegativeInteger)
	// for a simple declared type still compiles, while an unrelated ELEMENT-ONLY
	// complex alternative for a simple declared type is still rejected.
	t.Run("simple-to-simple still accepted", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="xs:nonNegativeInteger"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})

	t.Run("unrelated element-only complex still rejected", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Box"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="Box"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// A built-in LIST type is NOT derived from its atomic ITEM type, so a complex
	// simpleContent alternative based on the list type is NOT substitutable for an
	// element declared as the item type (XSDCTA-861-001).
	listVsItem := func(name, listType, itemType string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="L">
    <xs:simpleContent>
      <xs:extension base="xs:` + listType + `"><xs:attribute name="tag" type="xs:string"/></xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:` + itemType + `">
    <xs:alternative test="true()" type="L"/>
  </xs:element>
</xs:schema>`
			_, err := compile(t, s)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
	listVsItem("NMTOKENS list not substitutable for NMTOKEN item", "NMTOKENS", "NMTOKEN")
	listVsItem("IDREFS list not substitutable for IDREF item", "IDREFS", "IDREF")
	listVsItem("ENTITIES list not substitutable for ENTITY item", "ENTITIES", "ENTITY")

	// The legitimate atomic string-family chain (xs:NMTOKEN ⊂ xs:token) is still
	// accepted, confirming the list fix did not over-tighten atomic derivations.
	t.Run("complex simpleContent extending NMTOKEN accepted for declared token", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="NM">
    <xs:simpleContent>
      <xs:extension base="xs:NMTOKEN"><xs:attribute name="tag" type="xs:string"/></xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:token">
    <xs:alternative test="true()" type="NM"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})
}

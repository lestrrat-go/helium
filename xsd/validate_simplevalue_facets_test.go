package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// validateInstance compiles schemaXML and validates instanceXML against it,
// returning the validation error and collected error text.
func validateInstance(t *testing.T, schemaXML, instanceXML string) (error, string) {
	t.Helper()

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
	return err, errs
}

// TestListValueFacets verifies that list-level enumeration and pattern facets
// apply to the whole-list value (C-006). A list restricted to enumeration
// "1 2" must reject "3 4", and a list-level pattern must be enforced.
func TestListValueFacets(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="intList">
        <xs:enumeration value="1 2"/>
        <xs:enumeration value="9 8 7"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("enumeration member accepted", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<root>1 2</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("enumeration member with extra whitespace accepted", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<root>  9   8 7 </root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("enumeration non-member rejected", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<root>3 4</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'enumeration']")
	})

	t.Run("list pattern enforced", func(t *testing.T) {
		t.Parallel()
		const patternSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="tokList">
    <xs:list itemType="xs:token"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="tokList">
        <xs:pattern value="[a-z]+ [a-z]+"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		err, errs := validateInstance(t, patternSchema, `<root>foo bar</root>`)
		require.NoError(t, err, "validation errors: %s", errs)

		err, errs = validateInstance(t, patternSchema, `<root>foo 99</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'pattern']")
	})
}

// TestUnionEnumerationValueSpace verifies that union-level enumeration is
// compared in the active member's value space (C-007). A union of xs:int with
// enumeration "5" must accept "+5" (value-equal to 5), not reject it lexically.
func TestUnionEnumerationValueSpace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrName">
    <xs:union memberTypes="xs:int xs:NCName"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="intOrName">
        <xs:enumeration value="5"/>
        <xs:enumeration value="foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("value-equal int accepted", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<root>+5</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("exact int member accepted", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<root>5</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("string member accepted", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<root>foo</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("non-member rejected", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<root>7</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "is not a valid value")
	})
}

// TestQNameUnboundPrefix verifies that an xs:QName value with an unbound prefix
// is rejected (C-008). Lexical NCName form alone is not sufficient: the prefix
// must be bound in scope.
func TestQNameUnboundPrefix(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="q" type="xs:QName"/>
</xs:schema>`

	t.Run("unbound prefix rejected", func(t *testing.T) {
		t.Parallel()
		err, _ := validateInstance(t, schemaXML, `<q>p:foo</q>`)
		require.Error(t, err)
	})

	t.Run("bound prefix accepted", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<q xmlns:p="urn:p">p:foo</q>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("no prefix accepted", func(t *testing.T) {
		t.Parallel()
		err, errs := validateInstance(t, schemaXML, `<q>foo</q>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})
}

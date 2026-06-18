package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// validateInstance compiles schemaXML and validates instanceXML against it,
// returning the collected error text and the validation error.
func validateInstance(t *testing.T, schemaXML, instanceXML string) (string, error) {
	t.Helper()

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
	return errs, err
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
		errs, err := validateInstance(t, schemaXML, `<root>1 2</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("enumeration member with extra whitespace accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>  9   8 7 </root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("enumeration non-member rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>3 4</root>`)
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
		errs, err := validateInstance(t, patternSchema, `<root>foo bar</root>`)
		require.NoError(t, err, "validation errors: %s", errs)

		errs, err = validateInstance(t, patternSchema, `<root>foo 99</root>`)
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
		errs, err := validateInstance(t, schemaXML, `<root>+5</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("exact int member accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>5</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("string member accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>foo</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("non-member rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>7</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "is not a valid value")
	})
}

// TestQNameListNamespaceContext verifies that a list whose item type is xs:QName
// validates each item against the instance's in-scope namespaces. A bound prefix
// must be accepted and an unbound prefix rejected — the list path must thread the
// namespace context down to each item (regression: items were validated with a
// nil namespace map, wrongly rejecting valid bound QName items).
func TestQNameListNamespaceContext(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qnameList">
    <xs:list itemType="xs:QName"/>
  </xs:simpleType>
  <xs:element name="root" type="qnameList"/>
</xs:schema>`

	t.Run("bound prefixes accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root xmlns:p="urn:p" xmlns:q="urn:q">p:a q:b</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("single bound prefix accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root xmlns:p="urn:p">p:a</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("unbound prefix rejected", func(t *testing.T) {
		t.Parallel()
		_, err := validateInstance(t, schemaXML, `<root xmlns:p="urn:p">p:a q:b</root>`)
		require.Error(t, err)
	})
}

// TestListEnumerationValueSpace verifies that a list-level enumeration is compared
// in the item type's VALUE space, not raw collapsed lexical text. A list of xs:int
// with enumeration "1 2" must accept the value-equal instance "01 +2", and a list
// of xs:QName must compare items namespace-aware.
func TestListEnumerationValueSpace(t *testing.T) {
	t.Parallel()

	t.Run("int list value-equal accepted", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="intList">
        <xs:enumeration value="1 2"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs, err := validateInstance(t, schemaXML, `<root>01 +2</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("int list value-distinct rejected", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="intList">
        <xs:enumeration value="1 2"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs, err := validateInstance(t, schemaXML, `<root>1 3</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'enumeration']")
	})

	t.Run("qname list namespace-aware enumeration accepted", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:e="urn:e">
  <xs:simpleType name="qnameList">
    <xs:list itemType="xs:QName"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="qnameList">
        <xs:enumeration value="e:a"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		// Instance binds a DIFFERENT prefix to the same URI as the enumeration
		// literal — value-space equality requires resolving both to {urn:e}a.
		errs, err := validateInstance(t, schemaXML, `<root xmlns:p="urn:e">p:a</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})
}

// TestUnionEnumerationCrossMember verifies that a union-level enumeration resolves
// the active member INDEPENDENTLY for the instance value and for each enumeration
// literal, then compares with ordered-union value-family semantics. A look-alike in
// a different member must NOT be accepted: with memberTypes="zeroString xs:int" and
// enumeration "0" (active in the string member), the instance "+0" (active in
// xs:int) must be rejected even though both look numeric.
func TestUnionEnumerationCrossMember(t *testing.T) {
	t.Parallel()

	// zeroString is an xs:string restricted to exactly "0", so the literal "0" is
	// active in the string member; "+0" is not a valid zeroString, so it falls to
	// the xs:int member.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="zeroString">
    <xs:restriction base="xs:string">
      <xs:enumeration value="0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="zeroStringOrInt">
    <xs:union memberTypes="zeroString xs:int"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="zeroStringOrInt">
        <xs:enumeration value="0"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("exact string member literal accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>0</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("cross-member look-alike rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>+0</root>`)
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
		_, err := validateInstance(t, schemaXML, `<q>p:foo</q>`)
		require.Error(t, err)
	})

	t.Run("bound prefix accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<q xmlns:p="urn:p">p:foo</q>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("no prefix accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<q>foo</q>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})
}

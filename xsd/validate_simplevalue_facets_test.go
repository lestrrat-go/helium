package xsd_test

import (
	"strings"
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

	// Positive cross-member case: the enumeration literal and the instance value
	// resolve to DIFFERENT active members yet are value-equal across the
	// ordered-union value family, so the instance must be ACCEPTED. With
	// memberTypes="xs:int xs:decimal" the literal "1.0" is active in xs:decimal
	// (not a valid xs:int) while the instance "1" is active in xs:int; their value
	// spaces overlap, so 1 == 1.0 and the value matches. A naive implementation
	// that rejected every differing active member would wrongly fail this.
	t.Run("cross-member value-equal accepted", func(t *testing.T) {
		t.Parallel()
		const xsd = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrDecimal">
    <xs:union memberTypes="xs:int xs:decimal"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="intOrDecimal">
        <xs:enumeration value="1.0"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs, err := validateInstance(t, xsd, `<root>1</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})
}

// compileSchemaErrors compiles schemaXML with a label so that schema-component
// checks are active, and returns the collected compile error text. A non-empty
// result means the schema was rejected at compile time.
func compileSchemaErrors(t *testing.T, schemaXML string) string {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, _ = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	_ = collector.Close()

	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return b.String()
}

// TestNotationEnumeration exercises the bounded xs:NOTATION conformance: (a) an
// enumeration-derived NOTATION type resolves enumeration literal prefixes through
// the shared QName prefix-binding path, and an instance with an unbound prefix is
// rejected while a bound one is accepted; (b) a simpleType whose base is directly
// xs:NOTATION with no enumeration facet is rejected at compile time. Full
// xs:NOTATION declaration-table semantics (matching enumerated names against
// declared <xs:notation> elements) is deferred (see memory).
func TestNotationEnumeration(t *testing.T) {
	t.Parallel()

	const enumSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:notation name="png" public="image/png"/>
  <xs:simpleType name="imageNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
      <xs:enumeration value="p:png"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="n" type="imageNotation"/>
</xs:schema>`

	t.Run("enumeration-derived unbound prefix rejected", func(t *testing.T) {
		t.Parallel()
		_, err := validateInstance(t, enumSchema, `<n>q:jpeg</n>`)
		require.Error(t, err)
	})

	t.Run("enumeration-derived bound prefix accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, enumSchema, `<n xmlns:p="urn:p">p:jpeg</n>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("direct un-enumerated NOTATION base rejected at compile", func(t *testing.T) {
		t.Parallel()
		const bareSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="imageNotation">
    <xs:restriction base="xs:NOTATION"/>
  </xs:simpleType>
  <xs:element name="n" type="imageNotation"/>
</xs:schema>`
		errs := compileSchemaErrors(t, bareSchema)
		require.NotEmpty(t, errs, "expected a compile error for un-enumerated NOTATION base")
		require.Contains(t, errs, "NOTATION")
	})
}

// TestEnumQNameUnboundPrefixCompileError verifies item 1: an enumeration literal
// of a QName/NOTATION-restricted type whose prefix is not bound in the literal's
// in-scope namespaces is reported as a compile-time schema error, rather than
// silently compiling into an unsatisfiable enumeration.
func TestEnumQNameUnboundPrefixCompileError(t *testing.T) {
	t.Parallel()

	t.Run("unbound prefix rejected at compile", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction base="xs:QName">
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for unbound QName enumeration prefix")
		require.Contains(t, errs, "p:foo")
	})

	t.Run("bound prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction base="xs:QName">
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for bound QName enumeration prefix")
	})
}

// TestEnumQNameUnboundPrefixVarietyAware verifies item 1: the enumeration-literal
// QName/NOTATION prefix validation is variety-aware. A list whose item type is
// xs:QName and a union with an xs:QName member must each reject an enumeration
// literal whose prefix is unbound, recursing through the list item type and
// resolving the union member the literal belongs to. A bound prefix compiles
// cleanly through both varieties.
func TestEnumQNameUnboundPrefixVarietyAware(t *testing.T) {
	t.Parallel()

	t.Run("list itemType=QName unbound prefix rejected at compile", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:list itemType="xs:QName"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for unbound QName list enumeration prefix")
		require.Contains(t, errs, "p:foo")
	})

	t.Run("list itemType=QName bound prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:list itemType="xs:QName"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for bound QName list enumeration prefix")
	})

	t.Run("union memberTypes QName int unbound prefix rejected at compile", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="xs:QName xs:int"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for unbound QName union enumeration prefix")
		require.Contains(t, errs, "p:foo")
	})

	t.Run("union memberTypes QName int bound prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="xs:QName xs:int"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for bound QName union enumeration prefix")
	})

	t.Run("union int member literal compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="xs:QName xs:int"/>
        </xs:simpleType>
        <xs:enumeration value="5"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for an int union enumeration literal")
	})
}

// TestEnumQNameUnboundPrefixRecursiveCarrier verifies item 1 (W09): the
// enumeration-literal QName/NOTATION prefix-binding check detects a QName/NOTATION
// carrier that is hidden inside a union member that is itself a LIST or a NESTED
// UNION — not only a directly-atomic QName/NOTATION member. An unbound prefix in
// such a literal must be a compile-time schema error; a bound prefix compiles
// cleanly.
func TestEnumQNameUnboundPrefixRecursiveCarrier(t *testing.T) {
	t.Parallel()

	t.Run("union-of-list QName member unbound prefix rejected", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qnameList">
    <xs:list itemType="xs:QName"/>
  </xs:simpleType>
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="qnameList xs:int"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for unbound QName in union-of-list enumeration")
		require.Contains(t, errs, "p:foo")
	})

	t.Run("union-of-list QName member bound prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:simpleType name="qnameList">
    <xs:list itemType="xs:QName"/>
  </xs:simpleType>
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="qnameList xs:int"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for bound QName in union-of-list enumeration: %s", errs)
	})

	t.Run("nested-union QName member unbound prefix rejected", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="innerUnion">
    <xs:union memberTypes="xs:QName xs:boolean"/>
  </xs:simpleType>
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="innerUnion xs:int"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for unbound QName in nested-union enumeration")
		require.Contains(t, errs, "p:foo")
	})

	t.Run("nested-union QName member bound prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:simpleType name="innerUnion">
    <xs:union memberTypes="xs:QName xs:boolean"/>
  </xs:simpleType>
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="innerUnion xs:int"/>
        </xs:simpleType>
        <xs:enumeration value="p:foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for bound QName in nested-union enumeration: %s", errs)
	})
}

// TestUnionOfListsEnumerationValueSpace verifies item 2 (W09): a union-level
// enumeration whose active member is itself a LIST compares item-by-item in the
// list's item-type value space. With memberTypes="intList decimalList" and
// enumeration "1.0 2.0" (active in decimalList), the value-equal instance "1 2"
// (active in intList) must be ACCEPTED — the cross-member comparison must split
// the lists and compare items in the shared decimal value space, not value-compare
// the whole multi-token strings as scalars.
func TestUnionOfListsEnumerationValueSpace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="decimalList">
    <xs:list itemType="xs:decimal"/>
  </xs:simpleType>
  <xs:simpleType name="intOrDecimalList">
    <xs:union memberTypes="intList decimalList"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="intOrDecimalList">
        <xs:enumeration value="1.0 2.0"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("cross-member value-equal list accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>1 2</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("exact decimal list literal accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>1.0 2.0</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("value-distinct list rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>1 3</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "is not a valid value")
	})

	// A list whose item type is itself a UNION (list-of-union) must still recurse
	// through the union to resolve each item's active member before comparing it
	// against the sibling decimal-list item. memberTypes="intList decimalList"
	// where intList's item type is intOrBool (union of xs:int / xs:boolean): the
	// enumeration "1.0 2.0" rejects intList (1.0 is neither int nor boolean) so it
	// is active in decimalList, while the instance "1 2" is active in intList. Each
	// item "1"/"2" must resolve through the union to its xs:int member and compare
	// equal to the decimal "1.0"/"2.0" — the comparator cannot drop the union item
	// variety. (xs:boolean, not xs:string, is used so the decimal literals do not
	// fall back to a string member and absorb intList first.)
	t.Run("list-of-union item recurses through the union", func(t *testing.T) {
		t.Parallel()
		const listOfUnionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:int xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="intList">
    <xs:list itemType="intOrBool"/>
  </xs:simpleType>
  <xs:simpleType name="decimalList">
    <xs:list itemType="xs:decimal"/>
  </xs:simpleType>
  <xs:simpleType name="intOrDecimalList">
    <xs:union memberTypes="intList decimalList"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="intOrDecimalList">
        <xs:enumeration value="1.0 2.0"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

		t.Run("value-equal list accepted", func(t *testing.T) {
			t.Parallel()
			errs, err := validateInstance(t, listOfUnionSchema, `<root>1 2</root>`)
			require.NoError(t, err, "validation errors: %s", errs)
		})

		t.Run("value-distinct list rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := validateInstance(t, listOfUnionSchema, `<root>1 3</root>`)
			require.Error(t, err)
			require.Contains(t, errs, "is not a valid value")
		})
	})

	// Deeper nesting: a union member that is itself a list-of-union
	// (union-of-list-of-union). The enumeration "1.0 2.0" is active in
	// decimalList; the instance "1 2" must descend outer union -> listOfUnion
	// member -> per-item intOrBool union -> xs:int before comparing to the
	// decimals, proving the recursion bottoms out at every level.
	t.Run("union of list-of-union recurses to the bottom", func(t *testing.T) {
		t.Parallel()
		const deepSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:int xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="listOfUnion">
    <xs:list itemType="intOrBool"/>
  </xs:simpleType>
  <xs:simpleType name="decimalList">
    <xs:list itemType="xs:decimal"/>
  </xs:simpleType>
  <xs:simpleType name="deepUnion">
    <xs:union memberTypes="listOfUnion decimalList"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="deepUnion">
        <xs:enumeration value="1.0 2.0"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

		t.Run("value-equal list accepted", func(t *testing.T) {
			t.Parallel()
			errs, err := validateInstance(t, deepSchema, `<root>1 2</root>`)
			require.NoError(t, err, "validation errors: %s", errs)
		})

		t.Run("value-distinct list rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := validateInstance(t, deepSchema, `<root>1 3</root>`)
			require.Error(t, err)
			require.Contains(t, errs, "is not a valid value")
		})
	})
}

// TestNotationCarrierRecursive verifies item 3 (W09): the NOTATION-without-
// enumeration check is recursive over varieties. An xs:list itemType="xs:NOTATION"
// and a union member typed xs:NOTATION (neither enumeration-derived) are rejected
// at compile time, while a list/union built over an enumeration-derived NOTATION
// type compiles cleanly.
func TestNotationCarrierRecursive(t *testing.T) {
	t.Parallel()

	t.Run("list of un-enumerated NOTATION rejected", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="notationList">
    <xs:list itemType="xs:NOTATION"/>
  </xs:simpleType>
  <xs:element name="n" type="notationList"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for list of un-enumerated NOTATION")
		require.Contains(t, errs, "NOTATION")
	})

	t.Run("union member un-enumerated NOTATION rejected", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="notationOrInt">
    <xs:union memberTypes="xs:NOTATION xs:int"/>
  </xs:simpleType>
  <xs:element name="n" type="notationOrInt"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for union member un-enumerated NOTATION")
		require.Contains(t, errs, "NOTATION")
	})

	t.Run("list of enumeration-derived NOTATION compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:simpleType name="imageNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="notationList">
    <xs:list itemType="imageNotation"/>
  </xs:simpleType>
  <xs:element name="n" type="notationList"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for list of enumeration-derived NOTATION: %s", errs)
	})

	t.Run("union member enumeration-derived NOTATION compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:simpleType name="imageNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="notationOrInt">
    <xs:union memberTypes="imageNotation xs:int"/>
  </xs:simpleType>
  <xs:element name="n" type="notationOrInt"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for union member enumeration-derived NOTATION: %s", errs)
	})
}

// TestNotationTypeOnDeclaration verifies item 2: typing an element or attribute
// directly as xs:NOTATION (no enumeration facet) is rejected at compile time,
// mirroring the simpleType-level rule that xs:NOTATION may only be used in a
// derivation that supplies an enumeration of permitted notation names. An
// enumeration-derived NOTATION type used the same way compiles cleanly.
func TestNotationTypeOnDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("element type=NOTATION rejected at compile", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="n" type="xs:NOTATION"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for element type=xs:NOTATION")
		require.Contains(t, errs, "NOTATION")
	})

	t.Run("attribute type=NOTATION rejected at compile", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:NOTATION"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for attribute type=xs:NOTATION")
		require.Contains(t, errs, "NOTATION")
	})

	t.Run("element typed via enumeration-derived NOTATION compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:simpleType name="imageNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="n" type="imageNotation"/>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="imageNotation"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for an enumeration-derived NOTATION type: %s", errs)
	})
}

// TestQNamePredeclaredXMLPrefix verifies that the predeclared "xml" prefix is
// always in scope for xs:QName values without an explicit namespace declaration.
// The prefix-binding check must special-case "xml" (bound to the XML namespace);
// otherwise a value like "xml:lang" is wrongly rejected as unbound. Coverage spans
// a plain QName, a QName list item, and a QName enumeration member.
func TestQNamePredeclaredXMLPrefix(t *testing.T) {
	t.Parallel()

	t.Run("plain qname xml prefix accepted", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="q" type="xs:QName"/>
</xs:schema>`
		errs, err := validateInstance(t, schemaXML, `<q>xml:lang</q>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("qname list xml prefix item accepted", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qnameList">
    <xs:list itemType="xs:QName"/>
  </xs:simpleType>
  <xs:element name="root" type="qnameList"/>
</xs:schema>`
		errs, err := validateInstance(t, schemaXML, `<root>xml:lang xml:space</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("qname enumeration xml prefix member accepted", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xml="http://www.w3.org/XML/1998/namespace">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction base="xs:QName">
        <xs:enumeration value="xml:lang"/>
        <xs:enumeration value="xml:space"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		errs, err := validateInstance(t, schemaXML, `<q>xml:lang</q>`)
		require.NoError(t, err, "validation errors: %s", errs)

		errs, err = validateInstance(t, schemaXML, `<q>xml:base</q>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'enumeration']")
	})
}

// TestDerivedUnionRestrictionInheritsFacets verifies that a restriction derived
// from an already-enumerated union inherits the base union's facets even when it
// adds none of its own. With onlyFive = restriction(intOrString, enumeration="5")
// and derivedOnlyFive = restriction(onlyFive) (no new facets), an instance "7"
// must be rejected — validateUnionValue must walk the restriction base chain.
func TestDerivedUnionRestrictionInheritsFacets(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrString">
    <xs:union memberTypes="xs:int xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="onlyFive">
    <xs:restriction base="intOrString">
      <xs:enumeration value="5"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="derivedOnlyFive">
    <xs:restriction base="onlyFive"/>
  </xs:simpleType>
  <xs:element name="root" type="derivedOnlyFive"/>
</xs:schema>`

	t.Run("inherited enumeration member accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>5</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("inherited enumeration non-member rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>7</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "is not a valid value")
	})
}

// TestUnionRangeFacetRejectedAtCompile verifies that a range facet (minInclusive)
// on a UNION variety is rejected at COMPILE time, regardless of the members'
// types. Range facets are applicable only to atomic ordered value spaces; a
// union's value space is not ordered (XSD §4.1.5 {applicable facets}), so even a
// union of ordered numeric members, or a union whose member is itself a list, is
// rejected — matching xmllint ("The facet 'minInclusive' is not allowed."). This
// is the convergence regression: silently accepting the facet here turned the
// range comparison into a no-op at validation time and let any instance through.
func TestUnionRangeFacetRejectedAtCompile(t *testing.T) {
	t.Parallel()

	t.Run("union whose member is a list", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="intList"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="u">
        <xs:minInclusive value="10"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileSchemaErrors(t, schemaXML),
			"local union type: The facet 'minInclusive' is not allowed.")
	})

	t.Run("union of an ordered numeric member", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union memberTypes="xs:int"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="u">
        <xs:minInclusive value="10"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileSchemaErrors(t, schemaXML),
			"local union type: The facet 'minInclusive' is not allowed.")
	})
}

// TestUnionNonOrderedRangeFacetRejected verifies that a range facet
// (minInclusive) on a union whose member is a NON-ordered atomic type (boolean,
// the binary types) is rejected at COMPILE time. Range facets are applicable only
// to ORDERED value spaces (numeric and the date/time/duration family); boolean
// and the binary types are NOT ordered even though value.Compare returns a
// deterministic total order for them (that order exists only so enumeration can
// use cmp==0). Because the facet is inapplicable to the union variety, xmllint
// rejects the schema outright — accepting it would make the range comparison a
// no-op at validation time and silently drop the constraint.
//
// The compareForRangeFacet gate (value.Orderable) still protects the
// VALIDATION path for a genuinely reachable non-ordered atomic value (e.g. a list
// item or an active union member at instance time); this test covers the
// COMPILE-time rejection of the inapplicable facet on the union restriction.
func TestUnionNonOrderedRangeFacetRejected(t *testing.T) {
	t.Parallel()

	t.Run("boolean union range facet is not allowed", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union memberTypes="xs:boolean"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="u">
        <xs:minInclusive value="1"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileSchemaErrors(t, schemaXML),
			"local union type: The facet 'minInclusive' is not allowed.")
	})

	t.Run("base64Binary union range facet is not allowed", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union memberTypes="xs:base64Binary"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="u">
        <xs:minInclusive value="Qg=="/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileSchemaErrors(t, schemaXML),
			"local union type: The facet 'minInclusive' is not allowed.")
	})
}

// TestOrderedNonNumericRangeFacet verifies the positive side of the ordered-type
// gate: a range facet on an ORDERED but NON-numeric type (xs:date) is still
// enforced. The date/time/duration family is ordered per XSD, so minInclusive
// must reject an earlier date and accept the bound and a later date.
func TestOrderedNonNumericRangeFacet(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="xs:date">
        <xs:minInclusive value="2000-01-01"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("earlier date rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>1999-12-31</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'minInclusive']")
	})

	t.Run("bound date accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>2000-01-01</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("later date accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>2001-06-06</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})
}

// TestDurationRangeFacetReferenceDateOrder verifies that xs:duration range
// facets use the XSD reference-date partial order instead of comparing the
// months and seconds components independently. P1M is greater than P27D at all
// four reference dates, while P1M vs P30D remains indeterminate and therefore
// does not fire the facet.
func TestDurationRangeFacetReferenceDateOrder(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="xs:duration">
        <xs:minInclusive value="P1M"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("determinate shorter duration rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>P27D</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'minInclusive']")
	})

	t.Run("indeterminate duration accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>P30D</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})
}

// TestNumericUnionRangeFacetRejectedAtCompile verifies that a range facet on a
// union is rejected at COMPILE time even when every union member is numeric
// (ordered). A union's value space is not a single ordered scalar, so per XSD
// §4.1.5 the range facets are not in its {applicable facets} set regardless of
// member ordering — checkFacetApplicability reports "not allowed" rather than
// letting the bound compile into a runtime no-op.
func TestNumericUnionRangeFacetRejectedAtCompile(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union memberTypes="xs:int xs:decimal"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="u">
        <xs:minInclusive value="10"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	errs := compileSchemaErrors(t, schemaXML)
	require.Contains(t, errs, "The facet 'minInclusive' is not allowed.")
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

// TestQNameUnprefixedIgnoresDefaultNamespace verifies that an UNPREFIXED
// xs:QName *value* resolves to NO namespace and never picks up the in-scope
// default namespace — for either the schema's enumeration literal or the
// instance value (unlike element/attribute names, a default namespace does not
// bind unprefixed QName values per XSD). The schema and instance deliberately
// declare DIFFERENT default namespaces; an enumeration of unprefixed "foo"
// must still match an unprefixed instance "foo" because both are {no-ns}foo.
func TestQNameUnprefixedIgnoresDefaultNamespace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="urn:schema-default"
    targetNamespace="urn:tns"
    xmlns:tns="urn:tns">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction base="xs:QName">
        <xs:enumeration value="foo"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("unprefixed value matches enumeration across different default namespaces", func(t *testing.T) {
		t.Parallel()
		// The instance's default namespace (urn:instance-default) differs from the
		// schema's (urn:schema-default). The root element is placed in the target
		// namespace via a prefix so it matches the global declaration; the
		// unprefixed QName value "foo" must resolve to {no-ns}foo on both sides
		// (ignoring both default namespaces), so the enumeration matches.
		errs, err := validateInstance(t, schemaXML,
			`<tns:q xmlns:tns="urn:tns" xmlns="urn:instance-default">foo</tns:q>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("unprefixed value does not pick up instance default namespace", func(t *testing.T) {
		t.Parallel()
		// A prefixed instance value bound to the instance default URI must NOT
		// match the unprefixed {no-ns}foo enumeration.
		errs, err := validateInstance(t, schemaXML,
			`<tns:q xmlns:tns="urn:tns" xmlns:d="urn:instance-default">d:foo</tns:q>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'enumeration']")
	})
}

// TestNotationUnprefixedIgnoresDefaultNamespace mirrors the QName case for
// xs:NOTATION: an UNPREFIXED NOTATION *value* resolves to NO namespace and must
// not pick up the in-scope default namespace of either the schema or the
// instance. The schema and instance declare DIFFERENT default namespaces, yet
// an enumeration of unprefixed "jpeg" must match an unprefixed instance "jpeg".
func TestNotationUnprefixedIgnoresDefaultNamespace(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="urn:schema-default"
    targetNamespace="urn:tns"
    xmlns:tns="urn:tns">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:notation name="png" public="image/png"/>
  <xs:element name="n">
    <xs:simpleType>
      <xs:restriction base="xs:NOTATION">
        <xs:enumeration value="jpeg"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("unprefixed value matches enumeration across different default namespaces", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML,
			`<tns:n xmlns:tns="urn:tns" xmlns="urn:instance-default">jpeg</tns:n>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("unprefixed value does not pick up instance default namespace", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML,
			`<tns:n xmlns:tns="urn:tns" xmlns:d="urn:instance-default">d:jpeg</tns:n>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'enumeration']")
	})
}

// TestEnumerationLiteralWhitespaceNormalized verifies that an enumeration facet
// literal is whitespace-normalized with the constrained type's effective
// whiteSpace facet BEFORE it is compared against the (already normalized)
// instance value. xs:token collapses internal runs of whitespace, so an
// enumeration literal "a  b" (two spaces) denotes the value "a b" and must match
// an instance "a b" — a raw lexical compare of the un-normalized literal would
// wrongly reject it.
func TestEnumerationLiteralWhitespaceNormalized(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="xs:token">
        <xs:enumeration value="a  b"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("token enumeration with collapsible literal accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>a b</root>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("token enumeration non-member still rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<root>a c</root>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'enumeration']")
	})
}

// TestEnumQNameLiteralWhitespaceNormalized verifies the QName side of the same
// normalization rule. An xs:QName enumeration literal " p:a " (surrounding
// spaces) is a value in the QName value space (whiteSpace=collapse), so its
// collapsed form "p:a" is a valid bound QName: it must NOT be reported as an
// invalid QName at compile time, and it must match the instance "p:a".
func TestEnumQNameLiteralWhitespaceNormalized(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:element name="q">
    <xs:simpleType>
      <xs:restriction base="xs:QName">
        <xs:enumeration value=" p:a "/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("surrounding-space QName literal compiles cleanly", func(t *testing.T) {
		t.Parallel()
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for whitespace-padded QName enumeration literal: %s", errs)
	})

	t.Run("surrounding-space QName literal matches instance", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<q xmlns:p="urn:p">p:a</q>`)
		require.NoError(t, err, "validation errors: %s", errs)
	})

	t.Run("non-member QName rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := validateInstance(t, schemaXML, `<q xmlns:p="urn:p">p:b</q>`)
		require.Error(t, err)
		require.Contains(t, errs, "[facet 'enumeration']")
	})
}

// TestNestedUnionRangeFacetRejectedAtCompile verifies that a range facet
// (minInclusive) on a union — flat or nested, over a string OR a numeric member —
// is rejected at COMPILE time. A union's value space is never a single ordered
// scalar, so the range facets are outside its {applicable facets} set (XSD
// §4.1.5) no matter what the member leaf resolves to; checkFacetApplicability
// reports "not allowed" rather than deferring an inapplicable bound to a runtime
// no-op (which previously let some instances through and wrongly rejected others).
func TestNestedUnionRangeFacetRejectedAtCompile(t *testing.T) {
	t.Parallel()

	t.Run("string-leaf nested union", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="inner">
    <xs:union memberTypes="xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="outer">
    <xs:union memberTypes="inner"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="outer">
        <xs:minInclusive value="10"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileSchemaErrors(t, schemaXML), "The facet 'minInclusive' is not allowed.")
	})

	t.Run("flat numeric union", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="numUnion">
    <xs:union memberTypes="xs:integer"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="numUnion">
        <xs:minInclusive value="10"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileSchemaErrors(t, schemaXML), "The facet 'minInclusive' is not allowed.")
	})

	t.Run("nested numeric union", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="inner">
    <xs:union memberTypes="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="outer">
    <xs:union memberTypes="inner"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="outer">
        <xs:minInclusive value="10"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileSchemaErrors(t, schemaXML), "The facet 'minInclusive' is not allowed.")
	})
}

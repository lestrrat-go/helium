package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestFacetValueAgainstBaseType verifies that a range-facet value
// (min/maxInclusive, min/maxExclusive) which is not a valid instance of the
// restricted base type's value space is reported as a fatal schema compile
// error rather than silently compiling into a no-op facet. Previously a bound
// such as <xs:minInclusive value="abc"/> on an xs:int base compiled cleanly and
// the constraint was dropped, so the type then accepted any int (e.g. -999).
func TestFacetValueAgainstBaseType(t *testing.T) {
	t.Parallel()

	const wantMsg = "is not a valid value of the base type"

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	t.Run("non-numeric minInclusive on xs:int", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:int">
      <xs:minInclusive value="abc"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("non-numeric maxInclusive on xs:int", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:int">
      <xs:maxInclusive value="xyz"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("non-numeric minExclusive on xs:decimal", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:decimal">
      <xs:minExclusive value="not-a-number"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("non-date maxExclusive on xs:date", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:date">
      <xs:maxExclusive value="oops"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("out-of-range value for xs:int subtype", func(t *testing.T) {
		t.Parallel()
		// 99999999999 overruns the xs:int value space, so the bound is not a
		// valid instance of the base type even though it is lexically numeric.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:int">
      <xs:maxInclusive value="99999999999"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("inline element simpleType with bad bound", func(t *testing.T) {
		t.Parallel()
		// An anonymous simpleType on an element never enters the named-type
		// table, so the bound check must reach it via the type-def source map.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="xs:int">
        <xs:minInclusive value="abc"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("inline attribute simpleType with bad bound", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="n">
        <xs:simpleType>
          <xs:restriction base="xs:int">
            <xs:maxInclusive value="xyz"/>
          </xs:restriction>
        </xs:simpleType>
      </xs:attribute>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	// The per-facet captured namespace context still exists for the (rare) case
	// of a prefixed bound on an ORDERED atomic base, but a range facet on a
	// QName-bearing base — atomic OR reached through a union — is never
	// applicable: QName is not an ordered primitive, so xmllint rejects the facet
	// outright. These cases were previously asserted to "compile cleanly" on the
	// false premise that the bound reached the value-space check; the applicability
	// rule makes them compile errors instead, matching xmllint.
	t.Run("range facet on QName-bearing union base is not allowed", func(t *testing.T) {
		t.Parallel()
		// Per xmllint a range facet on a union (its value space is not ordered) is
		// "not allowed", regardless of whether the bound looks like a valid QName.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:simpleType name="qn">
    <xs:union memberTypes="xs:QName"/>
  </xs:simpleType>
  <xs:simpleType name="bounded">
    <xs:restriction base="qn">
      <xs:minInclusive value="p:a"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), "The facet 'minInclusive' is not allowed.")
	})

	t.Run("range facet on QName atomic base is not allowed", func(t *testing.T) {
		t.Parallel()
		// QName is not an ordered primitive, so a range facet is inapplicable even
		// directly on xs:QName. xmllint names the offending primitive in the message.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:simpleType name="bounded">
    <xs:restriction base="xs:QName">
      <xs:maxInclusive value="p:a"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"The facet 'maxInclusive' is not allowed on types derived from the type xs:QName.")
	})

	// CONVERGENCE REGRESSION: a range facet on a union of an ORDERED and an
	// UNORDERED member must be rejected at compile time. Before the applicability
	// check, validateValue accepted the bound 'abc' because the xs:string member
	// accepted it, so the schema compiled and the range comparison became a no-op
	// at validation time — letting integer instances like -999 through. The fix
	// rejects the facet outright with xmllint's "not allowed" message.
	t.Run("minInclusive=abc on union(xs:int xs:string) is rejected and does not false-accept", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union memberTypes="xs:int xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="bounded">
    <xs:restriction base="u">
      <xs:minInclusive value="abc"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"union type 'bounded': The facet 'minInclusive' is not allowed.")
	})

	t.Run("range facet on list base is not allowed", func(t *testing.T) {
		t.Parallel()
		// A list value space is a sequence, not an ordered scalar, so a range facet
		// is inapplicable; the no-op comparison would otherwise drop the constraint.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="l">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="bounded">
    <xs:restriction base="l">
      <xs:maxInclusive value="5"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"list type 'bounded': The facet 'maxInclusive' is not allowed.")
	})

	t.Run("length facet on list base is allowed", func(t *testing.T) {
		t.Parallel()
		// minLength/maxLength/length ARE applicable to a list (measured as item
		// count), so a list restriction adding them must still compile cleanly.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="l">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="bounded">
    <xs:restriction base="l">
      <xs:minLength value="1"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.NotContains(t, compileErrors(t, schemaXML), "is not allowed")
	})

	t.Run("range facet on string atomic base is not allowed", func(t *testing.T) {
		t.Parallel()
		// A range facet on a non-ordered atomic primitive (string) is inapplicable;
		// the message names the primitive ancestor (xs:string for a token base).
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bounded">
    <xs:restriction base="xs:token">
      <xs:minInclusive value="abc"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"The facet 'minInclusive' is not allowed on types derived from the type xs:string.")
	})

	t.Run("digit facet on double atomic base is not allowed", func(t *testing.T) {
		t.Parallel()
		// totalDigits/fractionDigits apply only to the xs:decimal family; xs:double
		// (ordered, so range facets ARE fine) does not admit a digit notion.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bounded">
    <xs:restriction base="xs:double">
      <xs:totalDigits value="2"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"The facet 'totalDigits' is not allowed on types derived from the type xs:double.")
	})

	t.Run("range facet on date atomic base still compiles", func(t *testing.T) {
		t.Parallel()
		// xs:date is an ordered primitive, so a range facet remains applicable and
		// its bound is still checked for value-space validity (not rejected here).
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bounded">
    <xs:restriction base="xs:date">
      <xs:minInclusive value="2020-01-01"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.NotContains(t, compileErrors(t, schemaXML), "is not allowed")
	})

	t.Run("valid numeric bound still compiles", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ok">
    <xs:restriction base="xs:int">
      <xs:minInclusive value="0"/>
      <xs:maxInclusive value="100"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="ok"/>
</xs:schema>`
		require.NotContains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("length facet on numeric atomic base is not allowed", func(t *testing.T) {
		t.Parallel()
		// length/minLength/maxLength apply only to the string-derived, binary,
		// anyURI, QName and NOTATION primitives. On a numeric (decimal-family)
		// atomic such as xs:int the length facets are inapplicable; xmllint rejects
		// them naming the xs:decimal primitive ancestor.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:int">
      <xs:length value="3"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"The facet 'length' is not allowed on types derived from the type xs:decimal.")
	})

	t.Run("minLength facet on float atomic base is not allowed", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:float">
      <xs:minLength value="1"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"The facet 'minLength' is not allowed on types derived from the type xs:float.")
	})

	t.Run("length facet on date atomic base is not allowed", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:date">
      <xs:maxLength value="5"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"The facet 'maxLength' is not allowed on types derived from the type xs:date.")
	})

	t.Run("length facet on string atomic base is allowed", func(t *testing.T) {
		t.Parallel()
		// length IS applicable to string-derived primitives, so a string restriction
		// adding it must still compile cleanly.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ok">
    <xs:restriction base="xs:string">
      <xs:length value="3"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="ok"/>
</xs:schema>`
		require.NotContains(t, compileErrors(t, schemaXML), "is not allowed")
	})

	t.Run("length facet on hexBinary atomic base is allowed", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ok">
    <xs:restriction base="xs:hexBinary">
      <xs:length value="3"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="ok"/>
</xs:schema>`
		require.NotContains(t, compileErrors(t, schemaXML), "is not allowed")
	})

	t.Run("inconsistent date range bounds are rejected", func(t *testing.T) {
		t.Parallel()
		// minInclusive > maxInclusive in the xs:date value space is inconsistent;
		// xmllint rejects it. Previously the decimal-only comparison treated the
		// non-numeric date bounds as incomparable and let it compile.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:date">
      <xs:minInclusive value="2021-01-01"/>
      <xs:maxInclusive value="2020-01-01"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"It is an error for the value of 'minInclusive' to be greater than the value of 'maxInclusive'.")
	})

	t.Run("consistent date range bounds still compile", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ok">
    <xs:restriction base="xs:date">
      <xs:minInclusive value="2020-01-01"/>
      <xs:maxInclusive value="2021-01-01"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="ok"/>
</xs:schema>`
		require.NotContains(t, compileErrors(t, schemaXML), "It is an error for the value of")
	})

	t.Run("invalid int bounds report only the invalid-bound error not an extra ordering error", func(t *testing.T) {
		t.Parallel()
		// xs:int with minInclusive="1.5" maxInclusive="1.0": both bounds are invalid
		// in the xs:int value space, so they are reported by the bound-value check.
		// The same-type ordering comparison must NOT additionally fire a min>max
		// error: with a resolved builtin primitive, an incomparable (invalid) bound
		// pair is skipped rather than falling back to a lexical decimal comparison
		// that xmllint never performs.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:int">
      <xs:minInclusive value="1.5"/>
      <xs:maxInclusive value="1.0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		errs := compileErrors(t, schemaXML)
		require.Contains(t, errs, wantMsg, "the invalid bounds must still be reported")
		require.NotContains(t, errs,
			"It is an error for the value of 'minInclusive' to be greater than the value of 'maxInclusive'.",
			"no spurious ordering error when the bounds are invalid for the type")
	})

	t.Run("inconsistent dateTime exclusive bounds are rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:dateTime">
      <xs:minExclusive value="2021-01-01T00:00:00"/>
      <xs:maxExclusive value="2021-01-01T00:00:00"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML),
			"It is an error for the value of 'minExclusive' to be greater than or equal to the value of 'maxExclusive'.")
	})
}

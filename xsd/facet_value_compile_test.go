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

	t.Run("namespace-prefixed bound on QName-bearing union base resolves", func(t *testing.T) {
		t.Parallel()
		// The base is a union whose member is xs:QName, so the range-facet bound
		// "p:a" is a QName value whose prefix must be resolved using the in-scope
		// namespaces captured at the facet element. With the namespace context
		// threaded through, "p:a" resolves and the schema compiles cleanly;
		// previously the bound was validated with a nil namespace map, so the
		// resolvable prefix was wrongly reported invalid.
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
		require.NotContains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("unbound-prefix bound on QName-bearing union base still errors", func(t *testing.T) {
		t.Parallel()
		// The prefix "q" is not declared anywhere in scope, so the bound "q:a" is
		// not a valid QName and the bound-value check must still flag it.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:simpleType name="qn">
    <xs:union memberTypes="xs:QName"/>
  </xs:simpleType>
  <xs:simpleType name="bounded">
    <xs:restriction base="qn">
      <xs:minInclusive value="q:a"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("two range facets declaring different prefixes both resolve", func(t *testing.T) {
		t.Parallel()
		// minInclusive and maxInclusive each declare their OWN namespace prefix
		// (p: and q:) on their own facet element, with neither declared on an
		// ancestor. Each bound is a QName value that must be resolved with the
		// prefix in scope at its own element. The old shared-RangeNS code captured
		// only the first facet's namespaces, so the second bound (q:z) was
		// validated with the first facet's map — where q: is unbound — and was
		// wrongly reported invalid. With per-facet namespace context both resolve
		// and the schema compiles cleanly.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qn">
    <xs:union memberTypes="xs:QName"/>
  </xs:simpleType>
  <xs:simpleType name="bounded">
    <xs:restriction base="qn">
      <xs:minInclusive xmlns:p="urn:p" value="p:a"/>
      <xs:maxInclusive xmlns:q="urn:q" value="q:z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.NotContains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("per-facet prefix does not leak to sibling facet", func(t *testing.T) {
		t.Parallel()
		// minInclusive declares prefix p: on its own element; maxInclusive uses
		// prefix p: but does NOT declare it and no ancestor does either. The
		// binding must NOT leak from the sibling facet, so the maxInclusive bound
		// "p:z" is an unbound QName and must still be flagged.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qn">
    <xs:union memberTypes="xs:QName"/>
  </xs:simpleType>
  <xs:simpleType name="bounded">
    <xs:restriction base="qn">
      <xs:minInclusive xmlns:p="urn:p" value="p:a"/>
      <xs:maxInclusive value="p:z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bounded"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
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
}

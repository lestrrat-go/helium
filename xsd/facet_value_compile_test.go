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

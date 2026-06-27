package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11SubtypeFacetsValueSpace verifies that the XSD 1.1 date/time and
// duration subtypes (xs:dateTimeStamp, xs:dayTimeDuration, xs:yearMonthDuration)
// compare in their primitive value space for the enumeration and fixed facets, so
// a lexically distinct but value-equal instance is accepted. The union case
// further exercises primitiveValueSpaceFamily, which reduces a subtype to its
// primitive family (duration) so a cross-member fixed/enumeration comparison
// against a sibling xs:duration value succeeds.
func TestVersion11SubtypeFacetsValueSpace(t *testing.T) {
	const schemaDateTimeStamp = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:simpleType>
      <xs:restriction base="xs:dateTimeStamp">
        <xs:enumeration value="2020-01-01T00:00:00Z"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	const schemaDayTimeDuration = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:simpleType>
      <xs:restriction base="xs:dayTimeDuration">
        <xs:enumeration value="P1D" fixed="true"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	const schemaYearMonthDuration = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:simpleType>
      <xs:restriction base="xs:yearMonthDuration">
        <xs:enumeration value="P1Y"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	// Union crossing a 1.1 subtype (xs:dayTimeDuration) with its base family
	// (xs:duration): the enumeration literal "P0M" is an xs:duration value, and the
	// instance "PT0S" is a value-equal xs:dayTimeDuration. Accepting it requires
	// reducing both to the shared "duration" primitive family.
	const schemaUnion = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="xs:dayTimeDuration xs:duration"/>
        </xs:simpleType>
        <xs:enumeration value="P0M"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("dateTimeStamp enumeration accepts value-equal instance", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaDateTimeStamp, `<v>2019-12-31T19:00:00-05:00</v>`)
		require.NoError(t, err)
	})

	t.Run("dayTimeDuration fixed accepts value-equal instance", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaDayTimeDuration, `<v>PT24H</v>`)
		require.NoError(t, err)
	})

	t.Run("yearMonthDuration enumeration accepts value-equal instance", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaYearMonthDuration, `<v>P12M</v>`)
		require.NoError(t, err)
	})

	t.Run("union(dayTimeDuration,duration) enumeration crosses subtype family", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaUnion, `<v>PT0S</v>`)
		require.NoError(t, err)
	})
}

package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
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

func TestVersion11TemporalSubtypeRangeFacets(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="stamp">
          <xs:simpleType>
            <xs:restriction base="xs:dateTimeStamp">
              <xs:minInclusive value="2020-01-01T00:00:00Z"/>
              <xs:maxInclusive value="2020-12-31T23:59:59Z"/>
            </xs:restriction>
          </xs:simpleType>
        </xs:element>
        <xs:element name="day">
          <xs:simpleType>
            <xs:restriction base="xs:dayTimeDuration">
              <xs:minInclusive value="PT1H"/>
              <xs:maxInclusive value="P2D"/>
            </xs:restriction>
          </xs:simpleType>
        </xs:element>
        <xs:element name="month">
          <xs:simpleType>
            <xs:restriction base="xs:yearMonthDuration">
              <xs:minInclusive value="P1M"/>
              <xs:maxInclusive value="P2Y"/>
            </xs:restriction>
          </xs:simpleType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<root><stamp>2020-06-01T12:00:00Z</stamp><day>PT24H</day><month>P12M</month></root>`)
	require.NoError(t, err)

	err = compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<root><stamp>2021-01-01T00:00:00Z</stamp><day>PT24H</day><month>P12M</month></root>`)
	require.ErrorIs(t, err, xsd.ErrValidationFailed)
}

func TestVersion11TemporalExclusiveBoundRestriction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "direct base",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseStamp">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:maxExclusive value="2020-01-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="sameBound">
    <xs:restriction base="baseStamp">
      <xs:maxExclusive value="2019-12-31T19:00:00-05:00"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="v" type="sameBound"/>
</xs:schema>`,
		},
		{
			name: "inherited through intermediate",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseStamp">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:maxExclusive value="2020-01-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="intermediateStamp">
    <xs:restriction base="baseStamp">
      <xs:minInclusive value="2019-01-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="sameBound">
    <xs:restriction base="intermediateStamp">
      <xs:maxExclusive value="2019-12-31T19:00:00-05:00"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="v" type="sameBound"/>
</xs:schema>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), tt.schema,
				`<v>2019-12-31T23:00:00Z</v>`))
		})
	}
}

func TestVersion11TemporalExclusiveBoundRestrictionAgainstEffectiveBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "minExclusive equal to direct maxInclusive",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseStamp">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:maxInclusive value="2020-06-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="badStamp">
    <xs:restriction base="baseStamp">
      <xs:minExclusive value="2020-05-31T20:00:00-04:00"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
		{
			name: "minExclusive equal to inherited maxInclusive",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseStamp">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:maxInclusive value="2020-06-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="intermediateStamp">
    <xs:restriction base="baseStamp">
      <xs:minInclusive value="2020-01-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="badStamp">
    <xs:restriction base="intermediateStamp">
      <xs:minExclusive value="2020-05-31T20:00:00-04:00"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
		{
			name: "old minExclusive hidden by tighter minInclusive",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseStamp">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:minExclusive value="2020-01-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="tightenedStamp">
    <xs:restriction base="baseStamp">
      <xs:minInclusive value="2020-06-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="unrelatedStamp">
    <xs:restriction base="tightenedStamp">
      <xs:maxInclusive value="2020-12-31T23:59:59Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="badStamp">
    <xs:restriction base="unrelatedStamp">
      <xs:minExclusive value="2019-12-31T19:00:00-05:00"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
		{
			name: "old maxExclusive hidden by tighter maxInclusive",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseStamp">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:maxExclusive value="2020-12-31T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="tightenedStamp">
    <xs:restriction base="baseStamp">
      <xs:maxInclusive value="2020-06-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="unrelatedStamp">
    <xs:restriction base="tightenedStamp">
      <xs:minInclusive value="2020-01-01T00:00:00Z"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="badStamp">
    <xs:restriction base="unrelatedStamp">
      <xs:maxExclusive value="2020-12-30T19:00:00-05:00"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tt.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}

func TestVersion11FixedRangeFacetRestriction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "direct base",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseDuration">
    <xs:restriction base="xs:yearMonthDuration">
      <xs:minInclusive value="P1Y1M" fixed="true"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="badDuration">
    <xs:restriction base="baseDuration">
      <xs:minInclusive value="P1Y2M"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
		{
			name: "inherited through intermediate",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseDuration">
    <xs:restriction base="xs:yearMonthDuration">
      <xs:minInclusive value="P1Y1M" fixed="true"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="intermediateDuration">
    <xs:restriction base="baseDuration">
      <xs:maxInclusive value="P3Y"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="badDuration">
    <xs:restriction base="intermediateDuration">
      <xs:minInclusive value="P1Y2M"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tt.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}

func TestVersion11TemporalExplicitTimezoneFacet(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="tzRequired">
    <xs:restriction base="xs:dateTime">
      <xs:explicitTimezone value="required"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="tzProhibited">
    <xs:restriction base="xs:date">
      <xs:explicitTimezone value="prohibited"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="required" type="tzRequired"/>
  <xs:element name="prohibited" type="tzProhibited"/>
</xs:schema>`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<required>2020-01-01T00:00:00Z</required>`))
	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<required>2020-01-01T00:00:00</required>`), xsd.ErrValidationFailed)

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<prohibited>2020-01-01</prohibited>`))
	require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML,
		`<prohibited>2020-01-01Z</prohibited>`), xsd.ErrValidationFailed)
}

func TestVersion11ExplicitTimezoneFixedFacetRestriction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "fixed optional through intermediate",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="baseDateTime">
    <xs:restriction base="xs:dateTime">
      <xs:explicitTimezone value="optional" fixed="true"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="intermediateDateTime">
    <xs:restriction base="baseDateTime">
      <xs:minInclusive value="2020-01-01T00:00:00"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="badDateTime">
    <xs:restriction base="intermediateDateTime">
      <xs:explicitTimezone value="required"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
		{
			name: "invalid fixed lexical",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="badDateTime">
    <xs:restriction base="xs:dateTime">
      <xs:explicitTimezone value="required" fixed="maybe"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tt.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}

func TestVersion10IgnoresExplicitTimezoneFacet(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="tzRequired">
    <xs:restriction base="xs:dateTime">
      <xs:explicitTimezone value="required"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="v" type="tzRequired"/>
</xs:schema>`

	require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schemaXML,
		`<v>2020-01-01T00:00:00</v>`))
}

func TestVersion11DateTimeStampFixedBuiltInFacets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "explicitTimezone optional cannot relax dateTimeStamp",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:explicitTimezone value="optional"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
		{
			name: "whiteSpace replace cannot relax dateTimeStamp collapse",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="xs:dateTimeStamp">
      <xs:whiteSpace value="replace"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tt.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		})
	}
}

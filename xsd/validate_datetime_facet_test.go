package xsd_test

import (
	"fmt"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestDateTimeTimezoneFacetBounds covers min/max Inclusive and Exclusive facet
// checks where the facet value and the instance value differ in whether they
// carry a timezone.
//
// Per the XSD order relation (XSD 1.0 3.2.7.4), a non-timezoned value spans the
// instant interval [v-14:00, v+14:00]. When that whole interval lies below or
// above the timezoned operand, the comparison is determinate even though the
// timezone presence differs. Only an overlapping interval is indeterminate, and
// an indeterminate comparison passes the facet check.
func TestDateTimeTimezoneFacetBounds(t *testing.T) {
	t.Parallel()

	schemaFor := func(facet, value string) string {
		return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="xs:dateTime">
        <xs:%s value="%s"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`, facet, value)
	}

	cases := []struct {
		name     string
		facet    string
		bound    string
		instance string
		valid    bool
	}{
		// minInclusive bound with a timezone, no-TZ instance.
		{
			name:     "min determinate below rejects",
			facet:    "minInclusive",
			bound:    "2020-01-01T12:00:00Z",
			instance: "2019-12-30T00:00:00", // latest instant 2019-12-30T14:00Z still below bound
			valid:    false,
		},
		{
			name:     "min indeterminate accepts",
			facet:    "minInclusive",
			bound:    "2020-01-01T12:00:00Z",
			instance: "2020-01-01T00:00:00", // +/-14:00 interval straddles the bound
			valid:    true,
		},
		// maxInclusive bound with a timezone, no-TZ instance.
		{
			name:     "max determinate above rejects",
			facet:    "maxInclusive",
			bound:    "2020-01-01T12:00:00Z",
			instance: "2020-01-10T00:00:00", // earliest instant 2020-01-09T10:00Z still above bound
			valid:    false,
		},
		{
			name:     "max indeterminate accepts",
			facet:    "maxInclusive",
			bound:    "2020-01-01T12:00:00Z",
			instance: "2020-01-01T00:00:00", // +/-14:00 interval straddles the bound
			valid:    true,
		},
		// Mirror: no-TZ bound, timezoned instance.
		{
			name:     "min no-tz bound tz instance rejects",
			facet:    "minInclusive",
			bound:    "2020-01-01T12:00:00",
			instance: "2019-12-30T14:00:00Z", // determinately below the bound's earliest instant
			valid:    false,
		},
		{
			name:     "max no-tz bound tz instance rejects",
			facet:    "maxInclusive",
			bound:    "2020-01-01T12:00:00",
			instance: "2020-01-03T03:00:00Z", // determinately above the bound's latest instant
			valid:    false,
		},
		// Exclusive facets use the same comparison; a determinately out-of-range
		// no-TZ instance must still be rejected.
		{
			name:     "minExclusive determinate below rejects",
			facet:    "minExclusive",
			bound:    "2020-06-01T12:00:00Z",
			instance: "2020-05-30T00:00:00", // latest instant 2020-05-30T14:00Z still below bound
			valid:    false,
		},
		{
			name:     "maxExclusive determinate above rejects",
			facet:    "maxExclusive",
			bound:    "2020-06-01T12:00:00Z",
			instance: "2020-06-10T00:00:00", // earliest instant 2020-06-09T10:00Z still above bound
			valid:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaFor(tc.facet, tc.bound)))
			require.NoError(t, err)
			schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
			require.NoError(t, err)

			instance := fmt.Sprintf("<root>%s</root>", tc.instance)
			doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestVersion11YearZeroMixedTimezoneFacetBounds(t *testing.T) {
	t.Parallel()

	schemaFor := func(baseType, bound string) string {
		return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="xs:%s">
        <xs:maxInclusive value="%s"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`, baseType, bound)
	}

	cases := []struct {
		name     string
		baseType string
		bound    string
		instance string
	}{
		{
			name:     "dateTime year zero is full date",
			baseType: "dateTime",
			bound:    "0000-01-01T00:00:00Z",
			instance: "0000-01-10T00:00:00",
		},
		{
			name:     "date year zero is full date",
			baseType: "date",
			bound:    "0000-01-01Z",
			instance: "0000-01-10",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaFor(tc.baseType, tc.bound)))
			require.NoError(t, err)
			schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), schemaDOC)
			require.NoError(t, err)

			doc, err := helium.NewParser().Parse(t.Context(), fmt.Appendf(nil, "<root>%s</root>", tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
			require.Error(t, err)
			require.Contains(t, errs, "[facet 'maxInclusive']")
		})
	}
}

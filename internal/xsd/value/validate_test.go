package value_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/stretchr/testify/require"
)

// Test-fixture-only literals (no lexicon equivalent).
const (
	testAbc      = "abc"
	testP1Y      = "P1Y"
	test1foo     = "1foo"
	testUnderBar = "_bar"
	testDT0      = "2023-01-15T10:30:00"
	refDateTimeZ = "2020-01-01T12:00:00Z"
	testT0       = "10:30:00"
	testAB       = "a:b"
	testFoo      = "foo"

	typeGYear     = "gYear"
	typeGMonth    = "gMonth"
	typeGDay      = "gDay"
	typeGMonthDay = "gMonthDay"
)

func TestBuiltinTypeValidation(t *testing.T) {
	tests := []struct {
		typeName string
		valid    []string
		invalid  []string
	}{
		{
			typeName: lexicon.TypeFloat,
			valid:    []string{lexicon.XSLTVersion10, "-1.5", "+3.14", "1", ".5", "1.", "1e10", "1.5E-3", lexicon.FloatINF, lexicon.FloatNegINF, "+INF", lexicon.FloatNaN},
			invalid:  []string{"", testAbc, "1.2.3", "inf", "nan", "Inf"},
		},
		{
			typeName: lexicon.TypeDouble,
			valid:    []string{lexicon.XSLTVersion10, "-1.5", "1e10", lexicon.FloatINF, lexicon.FloatNegINF, lexicon.FloatNaN},
			invalid:  []string{"", testAbc, "inf", "nan"},
		},
		{
			typeName: lexicon.TypeDateTime,
			valid:    []string{testDT0, "2023-01-15T10:30:00Z", "2023-01-15T10:30:00.123", "2023-01-15T10:30:00+09:00", "2023-01-15T10:30:00-05:00", "-0001-01-01T00:00:00", "2023-01-15T23:59:59", "2024-01-01T24:00:00", "2024-01-01T00:00:00", "2024-01-01T12:00:00+14:00", "2024-01-01T12:00:00-14:00"},
			invalid:  []string{"", "2023-01-15", testT0, "2023-01-15 10:30:00", "2023-1-15T10:30:00", "2024-01-01T99:99:99", "2024-01-01T25:00:00", "2024-01-01T12:60:00", "2024-01-01T12:00:60", "2024-01-01T24:00:01", "2024-01-01T24:30:00", "2024-01-01T12:00:00+15:00", "2024-01-01T12:00:00+14:01"},
		},
		{
			typeName: lexicon.TypeTime,
			valid:    []string{testT0, "10:30:00Z", "10:30:00.123", "10:30:00+09:00", "10:30:00-05:00", "00:00:00", "23:59:59", "24:00:00", "24:00:00.0", "12:00:00+14:00", "12:00:00-14:00"},
			invalid:  []string{"", "10:30", testAbc, "99:99:99", "25:00:00", "12:60:00", "12:00:60", "24:00:01", "24:30:00", "24:00:00.5", "12:00:00+15:00", "12:00:00+14:01", "12:00:00+09:60"},
		},
		{
			typeName: lexicon.TypeDuration,
			valid:    []string{testP1Y, "P1M", "P1D", "PT1H", "PT1M", "PT1S", "P1Y2M3D", "P1Y2M3DT4H5M6S", "PT1.5S", "-P1Y", "P0Y"},
			invalid:  []string{"", "P", "PT", "1Y", "-P", "-PT", testAbc},
		},
		{
			typeName: typeGYear,
			valid:    []string{"2023", "-0001", "2023Z", "2023+09:00", "10000"},
			invalid:  []string{"", "23", testAbc, "2023-01"},
		},
		{
			typeName: "gYearMonth",
			valid:    []string{"2023-01", "2023-12", "-0001-06", "2023-01Z", "2023-01+09:00"},
			invalid:  []string{"", "2023", "2023-1", testAbc},
		},
		{
			typeName: typeGMonth,
			valid:    []string{"--01", "--12", "--06Z", "--06+09:00"},
			invalid:  []string{"", "-01", "01", testAbc},
		},
		{
			typeName: typeGMonthDay,
			valid:    []string{"--01-15", "--12-31", "--06-01Z", "--06-01+09:00"},
			invalid:  []string{"", "--0115", "-01-15", testAbc},
		},
		{
			typeName: typeGDay,
			valid:    []string{"---01", "---31", "---15Z", "---15+09:00"},
			invalid:  []string{"", "--01", "01", testAbc},
		},
		{
			typeName: "Name",
			valid:    []string{testFoo, testUnderBar, ":baz", "a.b", "a-b", testAB, "A123"},
			invalid:  []string{"", test1foo, ".foo", "-foo"},
		},
		{
			typeName: "NCName",
			valid:    []string{testFoo, testUnderBar, "a.b", "a-b", "A123"},
			invalid:  []string{"", test1foo, ".foo", "-foo", testAB, ":foo"},
		},
		{
			typeName: "ID",
			valid:    []string{testFoo, testUnderBar, "myId123"},
			invalid:  []string{"", test1foo, testAB},
		},
		{
			typeName: "IDREF",
			valid:    []string{testFoo, testUnderBar},
			invalid:  []string{"", test1foo, testAB},
		},
		{
			typeName: "ENTITY",
			valid:    []string{testFoo, testUnderBar},
			invalid:  []string{"", test1foo, testAB},
		},
		{
			typeName: "NMTOKEN",
			valid:    []string{testFoo, test1foo, ".foo", "-foo", testAB, "A.1-2:3_4"},
			invalid:  []string{"", "foo bar", "foo\ttab"},
		},
		{
			typeName: "normalizedString",
			valid:    []string{"hello world", "  spaces  ", ""},
			invalid:  []string{"has\ttab", "has\nnewline", "has\rreturn"},
		},
		{
			typeName: "token",
			valid:    []string{"hello world", "single", ""},
			invalid:  []string{"has\ttab", "has\nnewline", " leading", "trailing ", "double  space"},
		},
		{
			typeName: "base64Binary",
			valid:    []string{"", "SGVsbG8=", "AAAA", "AA==", "A "},
			invalid:  []string{"@@@", "SGVsbG8!"},
		},
		{
			typeName: "QName",
			valid:    []string{testFoo, "xs:string", testAB, "_foo:_bar"},
			invalid:  []string{"", test1foo, "a:b:c", ":foo"},
		},
		{
			typeName: "NOTATION",
			valid:    []string{testFoo, "ns:notation"},
			invalid:  []string{"", test1foo},
		},
		{
			typeName: "anyURI",
			valid:    []string{"", "http://example.com", "urn:isbn:123", "anything goes"},
			invalid:  []string{},
		},
		{
			typeName: "byte",
			valid:    []string{"-128", "0", "127", "+0"},
			invalid:  []string{"-129", "128", "999", testAbc},
		},
		{
			typeName: "short",
			valid:    []string{"-32768", "0", "32767"},
			invalid:  []string{"-32769", "32768"},
		},
		{
			typeName: "int",
			valid:    []string{"-2147483648", "0", "2147483647"},
			invalid:  []string{"-2147483649", "2147483648"},
		},
		{
			typeName: "long",
			valid:    []string{"-9223372036854775808", "0", "9223372036854775807"},
			invalid:  []string{"-9223372036854775809", "9223372036854775808"},
		},
		{
			typeName: "unsignedByte",
			valid:    []string{"0", "255"},
			invalid:  []string{"-1", "256"},
		},
		{
			typeName: "unsignedShort",
			valid:    []string{"0", "65535"},
			invalid:  []string{"-1", "65536"},
		},
		{
			typeName: "unsignedInt",
			valid:    []string{"0", "4294967295"},
			invalid:  []string{"-1", "4294967296"},
		},
		{
			typeName: "unsignedLong",
			valid:    []string{"0", "18446744073709551615"},
			invalid:  []string{"-1", "18446744073709551616"},
		},
		{
			typeName: "nonNegativeInteger",
			valid:    []string{"0", "1", "999999999999999999"},
			invalid:  []string{"-1", "-100"},
		},
		{
			typeName: "nonPositiveInteger",
			valid:    []string{"0", "-1", "-999999999999999999"},
			invalid:  []string{"1", "100"},
		},
		{
			typeName: "positiveInteger",
			valid:    []string{"1", "100", "999999999999999999"},
			invalid:  []string{"0", "-1"},
		},
		{
			typeName: "negativeInteger",
			valid:    []string{"-1", "-100", "-999999999999999999"},
			invalid:  []string{"0", "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			t.Parallel()
			for _, v := range tt.valid {
				err := value.ValidateBuiltin(v, tt.typeName)
				require.NoError(t, err, "type %s should accept %q", tt.typeName, v)
			}
			for _, v := range tt.invalid {
				err := value.ValidateBuiltin(v, tt.typeName)
				require.Error(t, err, "type %s should reject %q", tt.typeName, v)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		typ  string
		a, b string
		want int  // -1, 0, 1
		ok   bool // false means indeterminate
	}{
		// decimal
		{lexicon.TypeDecimal, lexicon.XSLTVersion10, "2.0", -1, true},
		{lexicon.TypeDecimal, "2.0", lexicon.XSLTVersion10, 1, true},
		{lexicon.TypeDecimal, "3.14", "3.14", 0, true},
		{lexicon.TypeDecimal, "-1", "1", -1, true},
		{lexicon.TypeDecimal, "0.5", "0.50", 0, true},

		// integer (uses decimal path)
		{"integer", "10", "20", -1, true},
		{"integer", "100", "100", 0, true},
		{"integer", "-5", "5", -1, true},

		// boolean — "true"/"1" and "false"/"0" are value-equal; invalid lexicals
		// are indeterminate.
		{lexicon.TypeBoolean, "true", "1", 0, true},
		{lexicon.TypeBoolean, "false", "0", 0, true},
		{lexicon.TypeBoolean, "true", "false", 1, true},
		{lexicon.TypeBoolean, "false", "true", -1, true},
		{lexicon.TypeBoolean, "maybe", "true", 0, false},

		// hexBinary — compared by decoded octets (bytes.Compare order), so case
		// is not significant and 0x0A < 0x0B.
		{"hexBinary", "0A", "0a", 0, true},
		{"hexBinary", "DEADbeef", "deadBEEF", 0, true},
		{"hexBinary", "0A", "0B", -1, true},
		{"hexBinary", "0G", "0A", 0, false}, // invalid hex -> indeterminate

		// base64Binary — compared by decoded octets, whitespace insignificant.
		{"base64Binary", "YWJj", "YW Jj", 0, true},
		{"base64Binary", "YWJj", "YWJk", -1, true}, // "abc" < "abd"
		{"base64Binary", "@@@@", "YWJj", 0, false}, // invalid -> indeterminate

		// float
		{lexicon.TypeFloat, lexicon.XSLTVersion10, "2.0", -1, true},
		{lexicon.TypeFloat, "2.0", lexicon.XSLTVersion10, 1, true},
		{lexicon.TypeFloat, "3.14", "3.14", 0, true},
		{lexicon.TypeFloat, lexicon.FloatINF, lexicon.XSLTVersion10, 1, true},
		{lexicon.TypeFloat, lexicon.FloatNegINF, lexicon.XSLTVersion10, -1, true},
		{lexicon.TypeFloat, lexicon.FloatNegINF, lexicon.FloatINF, -1, true},
		{lexicon.TypeFloat, lexicon.FloatINF, lexicon.FloatINF, 0, true},
		{lexicon.TypeFloat, lexicon.FloatNaN, lexicon.XSLTVersion10, 0, false},
		{lexicon.TypeFloat, lexicon.XSLTVersion10, lexicon.FloatNaN, 0, false},
		{lexicon.TypeFloat, lexicon.FloatNaN, lexicon.FloatNaN, 0, false},
		{lexicon.TypeFloat, "1e2", "100", 0, true},
		{lexicon.TypeFloat, "1.5E-3", "0.0015", 0, true},
		// xs:float value space is IEEE-754 single precision: 16777216 and
		// 16777217 round to the same float32 (2^24 and 2^24+1 are not both
		// representable), so they are equal in the xs:float value space.
		{lexicon.TypeFloat, "16777216", "16777217", 0, true},
		{lexicon.TypeFloat, "16777217", "16777216", 0, true},

		// double (the value space is float64, so 16777216 and 16777217 remain
		// distinct — only the float path rounds to single precision).
		{lexicon.TypeDouble, lexicon.FloatINF, lexicon.FloatNegINF, 1, true},
		{lexicon.TypeDouble, "1e10", "9999999999", 1, true},
		{lexicon.TypeDouble, "16777216", "16777217", -1, true},
		{lexicon.TypeDouble, "16777217", "16777216", 1, true},

		// dateTime
		{lexicon.TypeDateTime, testDT0, testDT0, 0, true},
		{lexicon.TypeDateTime, testDT0, "2023-01-16T10:30:00", -1, true},
		{lexicon.TypeDateTime, "2023-01-15T10:30:00Z", "2023-01-15T11:30:00+01:00", 0, true},
		{lexicon.TypeDateTime, "2023-01-15T10:30:00Z", testDT0, 0, false}, // mixed TZ, overlapping interval
		// Mixed TZ but determinate under the XSD 14-hour rule: the non-timezoned
		// operand's [v-14:00, v+14:00] interval lies entirely on one side.
		{lexicon.TypeDateTime, "2019-12-30T00:00:00", refDateTimeZ, -1, true}, // latest instant 2019-12-30T14:00Z < bound
		{lexicon.TypeDateTime, refDateTimeZ, "2019-12-30T00:00:00", 1, true},  // mirror of above
		{lexicon.TypeDateTime, "2020-01-10T00:00:00", refDateTimeZ, 1, true},  // earliest instant 2020-01-09T10:00Z > bound
		{lexicon.TypeDateTime, refDateTimeZ, "2020-01-10T00:00:00", -1, true}, // mirror of above
		// Genuinely indeterminate: ±14:00 interval straddles the bound.
		{lexicon.TypeDateTime, "2020-01-01T00:00:00", refDateTimeZ, 0, false},
		{lexicon.TypeDateTime, refDateTimeZ, "2020-01-01T00:00:00", 0, false},

		// date
		{"date", "2023-01-15", "2023-01-16", -1, true},
		{"date", "2023-01-15", "2023-01-15", 0, true},
		{"date", "2023-12-31", "2023-01-01", 1, true},
		{"date", "2023-01-15Z", "2023-01-15+00:00", 0, true},

		// time
		{lexicon.TypeTime, testT0, "11:30:00", -1, true},
		{lexicon.TypeTime, testT0, testT0, 0, true},
		{lexicon.TypeTime, "23:59:59", "00:00:00", 1, true},
		{lexicon.TypeTime, "10:30:00Z", "11:30:00+01:00", 0, true},

		// gYear
		{typeGYear, "2023", "2024", -1, true},
		{typeGYear, "2023", "2023", 0, true},
		{typeGYear, "-0001", "2023", -1, true},

		// gYearMonth
		{"gYearMonth", "2023-01", "2023-02", -1, true},
		{"gYearMonth", "2023-06", "2023-06", 0, true},

		// gMonth
		{typeGMonth, "--01", "--02", -1, true},
		{typeGMonth, "--12", "--12", 0, true},
		{typeGMonth, "--12", "--01", 1, true},

		// gDay
		{typeGDay, "---01", "---02", -1, true},
		{typeGDay, "---15", "---15", 0, true},
		{typeGDay, "---31", "---01", 1, true},

		// gMonthDay
		{typeGMonthDay, "--01-15", "--01-16", -1, true},
		{typeGMonthDay, "--06-01", "--06-01", 0, true},
		{typeGMonthDay, "--12-31", "--01-01", 1, true},

		// Mixed-timezone partial gregorian types: the determinate ±14:00 rule
		// needs a full calendar date, so these stay indeterminate rather than
		// producing a wrong determinate result from normalizing a zero
		// year/month/day field.
		{typeGYear, "2020", "2020Z", 0, false},
		{"gYearMonth", "2020-06", "2020-06Z", 0, false},
		{typeGMonth, "--06", "--06Z", 0, false},
		{typeGDay, "---15", "---15Z", 0, false},
		{typeGMonthDay, "--06-15", "--06-15Z", 0, false},

		// Mixed-timezone date and time DO get the determinate ±14:00 rule: date
		// has a full calendar date, and compareTime assigns a reference date.
		{lexicon.TypeDate, "2019-12-01", "2020-01-01Z", -1, true}, // interval entirely below bound
		{lexicon.TypeDate, "2020-01-01", "2020-01-01Z", 0, false}, // ±14:00 straddles -> indeterminate
		{lexicon.TypeTime, "00:00:00", "20:00:00Z", -1, true},     // determinately earlier
		{lexicon.TypeTime, "12:00:00", "12:00:00Z", 0, false},     // ±14:00 straddles -> indeterminate

		// duration
		{lexicon.TypeDuration, testP1Y, "P2Y", -1, true},
		{lexicon.TypeDuration, testP1Y, testP1Y, 0, true},
		{lexicon.TypeDuration, "P2Y", testP1Y, 1, true},
		{lexicon.TypeDuration, "PT3600S", "PT1H", 0, true},
		{lexicon.TypeDuration, "P1D", "PT86400S", 0, true},
		{lexicon.TypeDuration, "-P1Y", testP1Y, -1, true},
		{lexicon.TypeDuration, "P1Y2M", "P1Y3M", -1, true},
		{lexicon.TypeDuration, "P1M", "P30D", 0, false}, // indeterminate: months vs days
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s/%s_vs_%s", tt.typ, tt.a, tt.b)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := value.Compare(tt.a, tt.b, tt.typ)
			require.Equal(t, tt.ok, ok, "ok mismatch")
			if ok {
				require.Equal(t, tt.want, got, "cmp mismatch")
			}
		})
	}
}

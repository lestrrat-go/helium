package xsd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinTypeValidation(t *testing.T) {
	tests := []struct {
		typeName string
		valid    []string
		invalid  []string
	}{
		{
			typeName: "float",
			valid:    []string{"1.0", "-1.5", "+3.14", "1", ".5", "1.", "1e10", "1.5E-3", "INF", "-INF", "+INF", "NaN"},
			invalid:  []string{"", "abc", "1.2.3", "inf", "nan", "Inf"},
		},
		{
			typeName: "double",
			valid:    []string{"1.0", "-1.5", "1e10", "INF", "-INF", "NaN"},
			invalid:  []string{"", "abc", "inf", "nan"},
		},
		{
			typeName: "dateTime",
			valid:    []string{"2023-01-15T10:30:00", "2023-01-15T10:30:00Z", "2023-01-15T10:30:00.123", "2023-01-15T10:30:00+09:00", "2023-01-15T10:30:00-05:00", "-0001-01-01T00:00:00"},
			invalid:  []string{"", "2023-01-15", "10:30:00", "2023-01-15 10:30:00", "2023-1-15T10:30:00"},
		},
		{
			typeName: "time",
			valid:    []string{"10:30:00", "10:30:00Z", "10:30:00.123", "10:30:00+09:00", "10:30:00-05:00", "00:00:00"},
			invalid:  []string{"", "10:30", "abc"},
		},
		{
			typeName: "duration",
			valid:    []string{"P1Y", "P1M", "P1D", "PT1H", "PT1M", "PT1S", "P1Y2M3D", "P1Y2M3DT4H5M6S", "PT1.5S", "-P1Y", "P0Y"},
			invalid:  []string{"", "P", "PT", "1Y", "-P", "-PT", "abc"},
		},
		{
			typeName: "gYear",
			valid:    []string{"2023", "-0001", "2023Z", "2023+09:00", "10000"},
			invalid:  []string{"", "23", "abc", "2023-01"},
		},
		{
			typeName: "gYearMonth",
			valid:    []string{"2023-01", "2023-12", "-0001-06", "2023-01Z", "2023-01+09:00"},
			invalid:  []string{"", "2023", "2023-1", "abc"},
		},
		{
			typeName: "gMonth",
			valid:    []string{"--01", "--12", "--06Z", "--06+09:00"},
			invalid:  []string{"", "-01", "01", "abc"},
		},
		{
			typeName: "gMonthDay",
			valid:    []string{"--01-15", "--12-31", "--06-01Z", "--06-01+09:00"},
			invalid:  []string{"", "--0115", "-01-15", "abc"},
		},
		{
			typeName: "gDay",
			valid:    []string{"---01", "---31", "---15Z", "---15+09:00"},
			invalid:  []string{"", "--01", "01", "abc"},
		},
		{
			typeName: "Name",
			valid:    []string{"foo", "_bar", ":baz", "a.b", "a-b", "a:b", "A123"},
			invalid:  []string{"", "1foo", ".foo", "-foo"},
		},
		{
			typeName: "NCName",
			valid:    []string{"foo", "_bar", "a.b", "a-b", "A123"},
			invalid:  []string{"", "1foo", ".foo", "-foo", "a:b", ":foo"},
		},
		{
			typeName: "ID",
			valid:    []string{"foo", "_bar", "myId123"},
			invalid:  []string{"", "1foo", "a:b"},
		},
		{
			typeName: "IDREF",
			valid:    []string{"foo", "_bar"},
			invalid:  []string{"", "1foo", "a:b"},
		},
		{
			typeName: "ENTITY",
			valid:    []string{"foo", "_bar"},
			invalid:  []string{"", "1foo", "a:b"},
		},
		{
			typeName: "NMTOKEN",
			valid:    []string{"foo", "1foo", ".foo", "-foo", "a:b", "A.1-2:3_4"},
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
			valid:    []string{"", "SGVsbG8=", "AAAA", "AA==", "A A A A"},
			invalid:  []string{"@@@", "SGVsbG8!"},
		},
		{
			typeName: "QName",
			valid:    []string{"foo", "xs:string", "a:b", "_foo:_bar"},
			invalid:  []string{"", "1foo", "a:b:c", ":foo"},
		},
		{
			typeName: "NOTATION",
			valid:    []string{"foo", "ns:notation"},
			invalid:  []string{"", "1foo"},
		},
		{
			typeName: "anyURI",
			valid:    []string{"", "http://example.com", "urn:isbn:123", "anything goes"},
			invalid:  []string{},
		},
		{
			typeName: "byte",
			valid:    []string{"-128", "0", "127", "+0"},
			invalid:  []string{"-129", "128", "999", "abc"},
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
				err := validateBuiltinValue(v, tt.typeName)
				require.NoError(t, err, "type %s should accept %q", tt.typeName, v)
			}
			for _, v := range tt.invalid {
				err := validateBuiltinValue(v, tt.typeName)
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
		{"decimal", "1.0", "2.0", -1, true},
		{"decimal", "2.0", "1.0", 1, true},
		{"decimal", "3.14", "3.14", 0, true},
		{"decimal", "-1", "1", -1, true},
		{"decimal", "0.5", "0.50", 0, true},

		// integer (uses decimal path)
		{"integer", "10", "20", -1, true},
		{"integer", "100", "100", 0, true},
		{"integer", "-5", "5", -1, true},

		// float
		{"float", "1.0", "2.0", -1, true},
		{"float", "2.0", "1.0", 1, true},
		{"float", "3.14", "3.14", 0, true},
		{"float", "INF", "1.0", 1, true},
		{"float", "-INF", "1.0", -1, true},
		{"float", "-INF", "INF", -1, true},
		{"float", "INF", "INF", 0, true},
		{"float", "NaN", "1.0", 0, false},
		{"float", "1.0", "NaN", 0, false},
		{"float", "NaN", "NaN", 0, false},
		{"float", "1e2", "100", 0, true},
		{"float", "1.5E-3", "0.0015", 0, true},

		// double (same path as float)
		{"double", "INF", "-INF", 1, true},
		{"double", "1e10", "9999999999", 1, true},

		// dateTime
		{"dateTime", "2023-01-15T10:30:00", "2023-01-15T10:30:00", 0, true},
		{"dateTime", "2023-01-15T10:30:00", "2023-01-16T10:30:00", -1, true},
		{"dateTime", "2023-01-15T10:30:00Z", "2023-01-15T11:30:00+01:00", 0, true},
		{"dateTime", "2023-01-15T10:30:00Z", "2023-01-15T10:30:00", 0, false}, // mixed TZ

		// date
		{"date", "2023-01-15", "2023-01-16", -1, true},
		{"date", "2023-01-15", "2023-01-15", 0, true},
		{"date", "2023-12-31", "2023-01-01", 1, true},
		{"date", "2023-01-15Z", "2023-01-15+00:00", 0, true},

		// time
		{"time", "10:30:00", "11:30:00", -1, true},
		{"time", "10:30:00", "10:30:00", 0, true},
		{"time", "23:59:59", "00:00:00", 1, true},
		{"time", "10:30:00Z", "11:30:00+01:00", 0, true},

		// gYear
		{"gYear", "2023", "2024", -1, true},
		{"gYear", "2023", "2023", 0, true},
		{"gYear", "-0001", "2023", -1, true},

		// gYearMonth
		{"gYearMonth", "2023-01", "2023-02", -1, true},
		{"gYearMonth", "2023-06", "2023-06", 0, true},

		// gMonth
		{"gMonth", "--01", "--02", -1, true},
		{"gMonth", "--12", "--12", 0, true},
		{"gMonth", "--12", "--01", 1, true},

		// gDay
		{"gDay", "---01", "---02", -1, true},
		{"gDay", "---15", "---15", 0, true},
		{"gDay", "---31", "---01", 1, true},

		// gMonthDay
		{"gMonthDay", "--01-15", "--01-16", -1, true},
		{"gMonthDay", "--06-01", "--06-01", 0, true},
		{"gMonthDay", "--12-31", "--01-01", 1, true},

		// duration
		{"duration", "P1Y", "P2Y", -1, true},
		{"duration", "P1Y", "P1Y", 0, true},
		{"duration", "P2Y", "P1Y", 1, true},
		{"duration", "PT3600S", "PT1H", 0, true},
		{"duration", "P1D", "PT86400S", 0, true},
		{"duration", "-P1Y", "P1Y", -1, true},
		{"duration", "P1Y2M", "P1Y3M", -1, true},
		{"duration", "P1M", "P30D", 0, false}, // indeterminate: months vs days
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s/%s_vs_%s", tt.typ, tt.a, tt.b)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := compareValues(tt.a, tt.b, tt.typ)
			require.Equal(t, tt.ok, ok, "ok mismatch")
			if ok {
				require.Equal(t, tt.want, got, "cmp mismatch")
			}
		})
	}
}

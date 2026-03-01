package xsd

import (
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

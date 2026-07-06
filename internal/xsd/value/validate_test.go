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
	testP1M      = "P1M"
	test1foo     = "1foo"
	testUnderBar = "_bar"
	testDT0      = "2023-01-15T10:30:00"
	testDate0    = "2023-01-15"
	testJan1     = "2023-01-01"
	refDateTimeZ = "2020-01-01T12:00:00Z"
	testT0       = "10:30:00"
	testAB       = "a:b"
	testFoo      = "foo"

	// Huge expanded-year fixtures: a 24-digit year exceeds an int and exercises
	// the arbitrary-precision year comparison path.
	hugeYear      = "999999999999999999999999"
	hugeYearPlus1 = "1000000000000000000000000"
	hugeDate      = hugeYear + "-01-01"
	hugeDatePlus1 = hugeYearPlus1 + "-01-01"

	typeGYear        = "gYear"
	typeGYearMonth   = "gYearMonth"
	typeGMonth       = "gMonth"
	typeGDay         = "gDay"
	typeGMonthDay    = "gMonthDay"
	typeBase64Binary = "base64Binary"
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
			// The optional sign applies to INF and the numeric forms but NOT to NaN:
			// "+NaN"/"-NaN" are not valid xs:float lexical forms.
			invalid: []string{"", testAbc, "1.2.3", "inf", "nan", "Inf", "+NaN", "-NaN"},
		},
		{
			typeName: lexicon.TypeDouble,
			valid:    []string{lexicon.XSLTVersion10, "-1.5", "1e10", "+INF", lexicon.FloatINF, lexicon.FloatNegINF, lexicon.FloatNaN},
			invalid:  []string{"", testAbc, "inf", "nan", "+NaN", "-NaN"},
		},
		{
			typeName: lexicon.TypeDateTime,
			valid:    []string{testDT0, "2023-01-15T10:30:00Z", "2023-01-15T10:30:00.123", "2023-01-15T10:30:00+09:00", "2023-01-15T10:30:00-05:00", "-0001-01-01T00:00:00", "2023-01-15T23:59:59", "2024-01-01T24:00:00", "2024-01-01T00:00:00", "2024-01-01T12:00:00+14:00", "2024-01-01T12:00:00-14:00"},
			invalid:  []string{"", testDate0, testT0, "2023-01-15 10:30:00", "2023-1-15T10:30:00", "2024-01-01T99:99:99", "2024-01-01T25:00:00", "2024-01-01T12:60:00", "2024-01-01T12:00:60", "2024-01-01T24:00:01", "2024-01-01T24:30:00", "2024-01-01T12:00:00+15:00", "2024-01-01T12:00:00+14:01"},
		},
		{
			typeName: lexicon.TypeTime,
			valid:    []string{testT0, "10:30:00Z", "10:30:00.123", "10:30:00+09:00", "10:30:00-05:00", "00:00:00", "23:59:59", "24:00:00", "24:00:00.0", "12:00:00+14:00", "12:00:00-14:00"},
			invalid:  []string{"", "10:30", testAbc, "99:99:99", "25:00:00", "12:60:00", "12:00:60", "24:00:01", "24:30:00", "24:00:00.5", "12:00:00+15:00", "12:00:00+14:01", "12:00:00+09:60"},
		},
		{
			typeName: lexicon.TypeDuration,
			valid:    []string{testP1Y, testP1M, "P1D", "PT1H", "PT1M", "PT1S", "P1Y2M3D", "P1Y2M3DT4H5M6S", "PT1.5S", "-P1Y", "P0Y"},
			invalid:  []string{"", "P", "PT", "1Y", "-P", "-PT", "P1YT", "P1DT", "P1Y2MT", testAbc},
		},
		{
			typeName: lexicon.TypeDate,
			// The huge-year cases exercise leap-year computation on expanded years
			// that overflow a fixed-width int: ...996 is divisible by 4 (leap),
			// ...999 is not (non-leap).
			valid:   []string{testDate0, "2000-02-29", "2400-02-29", "2023-12-31", "-0001-01-01", "2023-01-15Z", "2023-01-15+09:00", "2023-01-15-05:00", "2023-01-15+14:00", "2023-01-15-14:00", hugeDate, "999999999999999999999996-02-29"},
			invalid: []string{"", testAbc, "2023-13-01", "2023-00-01", "2023-01-32", "2023-01-00", "2001-02-29", "2100-02-29", "2023-01-15+99:99", "2023-01-15+15:00", "2023-01-15+14:01", "999999999999999999999999-02-29"},
		},
		{
			typeName: typeGYear,
			valid:    []string{"2023", "-0001", "2023Z", "2023+09:00", "10000"},
			invalid:  []string{"", "23", testAbc, "2023-01", "2023+99:99", "2023+15:00"},
		},
		{
			typeName: typeGYearMonth,
			valid:    []string{"2023-01", "2023-12", "-0001-06", "2023-01Z", "2023-01+09:00"},
			invalid:  []string{"", "2023", "2023-1", testAbc, "2023-13", "2023-00", "2023-01+99:99", "2023-01+15:00"},
		},
		{
			typeName: typeGMonth,
			valid:    []string{"--01", "--12", "--06Z", "--06+09:00"},
			invalid:  []string{"", "-01", "01", testAbc, "--13", "--00", "--06+99:99", "--06+15:00"},
		},
		{
			typeName: typeGMonthDay,
			valid:    []string{"--01-15", "--12-31", "--06-01Z", "--06-01+09:00", "--02-29"},
			invalid:  []string{"", "--0115", "-01-15", testAbc, "--13-01", "--00-01", "--01-32", "--01-00", "--02-30", "--06-01+99:99", "--06-01+15:00"},
		},
		{
			typeName: typeGDay,
			valid:    []string{"---01", "---31", "---15Z", "---15+09:00"},
			invalid:  []string{"", "--01", "01", testAbc, "---32", "---00", "---99", "---15+99:99", "---15+15:00"},
		},
		{
			typeName: "Name",
			valid:    []string{testFoo, testUnderBar, ":baz", "a.b", "a-b", testAB, "A123", "é", "naïve", "Ω"},
			// The 0xFF byte is malformed UTF-8; ranging it would yield U+FFFD, which
			// the Name char range admits, so it must be rejected as invalid.
			invalid: []string{"", test1foo, ".foo", "-foo", "a b", string([]byte{0xff}), "a" + string([]byte{0xff})},
		},
		{
			typeName: "NCName",
			valid:    []string{testFoo, testUnderBar, "a.b", "a-b", "A123", "é", "naïve", "Ω"},
			invalid:  []string{"", test1foo, ".foo", "-foo", testAB, ":foo", "a b", string([]byte{0xff}), "a" + string([]byte{0xff})},
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
			valid:    []string{testFoo, test1foo, ".foo", "-foo", testAB, "A.1-2:3_4", "café", "Ω"},
			invalid:  []string{"", "foo bar", "foo\ttab", string([]byte{0xff}), "a" + string([]byte{0xff})},
		},
		{
			typeName: "normalizedString",
			valid:    []string{"hello world", "  spaces  ", ""},
			invalid:  []string{"has\ttab", "has\nnewline", "has\rreturn"},
		},
		{
			// NBSP (U+00A0) is NOT XSD whitespace (only space/tab/CR/LF), so a
			// token may begin, end, or contain runs of NBSP: " abc "
			// has no leading/trailing XSD whitespace and no double ASCII space,
			// so it is a valid xs:token. Trimming with Go's Unicode TrimSpace
			// would wrongly strip the NBSP and reject it.
			typeName: "token",
			valid:    []string{"hello world", "single", "", " abc ", "a  b"},
			invalid:  []string{"has\ttab", "has\nnewline", " leading", "trailing ", "double  space"},
		},
		{
			typeName: typeBase64Binary,
			valid:    []string{"", "SGVsbG8=", "AAAA", "AA==", "AAAA ", " AA== ", "SGVs bG8="},
			// Form-feed is not XSD whitespace (only space/tab/CR/LF), so "TQ\f=="
			// must be rejected rather than treated as "TQ==". The "TR==", "AB==",
			// and "AAB=" forms carry non-zero unused trailing pad bits, which are
			// not valid xs:base64Binary lexical forms and must be rejected (strict
			// decode).
			invalid: []string{"@@@", "SGVsbG8!", "====", "A", "A===", "AAA", "TQ\f==", "TQ", "TR==", "AB==", "AAB="},
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
				err := value.ValidateBuiltin(v, tt.typeName, value.Version11)
				require.NoError(t, err, "type %s should accept %q", tt.typeName, v)
			}
			for _, v := range tt.invalid {
				err := value.ValidateBuiltin(v, tt.typeName, value.Version11)
				require.Error(t, err, "type %s should reject %q", tt.typeName, v)
			}
		})
	}
}

func TestAnyURIValidationVersionSpecific(t *testing.T) {
	valid10 := []string{"", "a", "urn:isbn:123", "http://example.com/a%20b", "../relative/path", "#frag", "日本語", "foo>bar", `foo"bar`, "has space"}
	for _, v := range valid10 {
		require.NoError(t, value.ValidateBuiltin(v, "anyURI", value.Version10), "XSD 1.0 anyURI %q", v)
	}

	invalid10 := []string{":a", "b:", "has\tcontrol", "bad%escape", "%", `\`, "^"}
	for _, v := range invalid10 {
		require.Error(t, value.ValidateBuiltin(v, "anyURI", value.Version10), "XSD 1.0 anyURI %q", v)
		require.NoError(t, value.ValidateBuiltin(v, "anyURI", value.Version11), "XSD 1.1 anyURI %q", v)
	}
}

func TestCalendarYearExpandedLexicalRejectsLeadingZeroes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ   string
		valid string
		bad   string
	}{
		{lexicon.TypeDate, "10000-01-01", "00000-01-01"},
		{lexicon.TypeDateTime, "10000-01-01T00:00:00", "00000-01-01T00:00:00"},
		{lexicon.TypeDateTimeStamp, "10000-01-01T00:00:00Z", "00000-01-01T00:00:00Z"},
		{typeGYear, "10000", "00000"},
		{typeGYearMonth, "10000-01", "00000-01"},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, value.ValidateBuiltin(tt.valid, tt.typ, value.Version11))
			require.Error(t, value.ValidateBuiltin(tt.bad, tt.typ, value.Version11))
			require.Error(t, value.ValidateBuiltin("-"+tt.bad, tt.typ, value.Version11))
			require.Error(t, value.ValidateBuiltin(tt.bad, tt.typ, value.Version10))
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
		{lexicon.TypeInteger, "10", "20", -1, true},
		{lexicon.TypeInteger, "100", "100", 0, true},
		{lexicon.TypeInteger, "-5", "5", -1, true},

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
		{typeBase64Binary, "YWJj", "YW Jj", 0, true},
		{typeBase64Binary, "YWJj", "YWJk", -1, true}, // "abc" < "abd"
		{typeBase64Binary, "@@@@", "YWJj", 0, false}, // invalid -> indeterminate
		// Unpadded operands are not valid base64Binary lexical forms, so a
		// comparison against them is indeterminate (no RawStdEncoding fallback).
		{typeBase64Binary, "TQ", "TQ==", 0, false},
		{typeBase64Binary, "TQ==", "TQ", 0, false},
		// "TR==" carries non-zero unused pad bits and is not a valid
		// xs:base64Binary lexical form, so it must decode-fail -> indeterminate.
		{typeBase64Binary, "TQ==", "TR==", 0, false},

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
		// "+NaN"/"-NaN" are not valid xs:float lexical forms, so a comparison
		// against them is indeterminate rather than treating them as NaN. This
		// keeps the value space consistent with the lexical validator.
		{lexicon.TypeFloat, "+NaN", lexicon.FloatNaN, 0, false},
		{lexicon.TypeFloat, "-NaN", lexicon.FloatNaN, 0, false},
		{lexicon.TypeFloat, lexicon.FloatNaN, "+NaN", 0, false},
		{lexicon.TypeFloat, "1e2", "100", 0, true},
		{lexicon.TypeFloat, "1.5E-3", "0.0015", 0, true},
		// xs:float value space is IEEE-754 single precision: 16777216 and
		// 16777217 round to the same float32 (2^24 and 2^24+1 are not both
		// representable), so they are equal in the xs:float value space.
		{lexicon.TypeFloat, "16777216", "16777217", 0, true},
		{lexicon.TypeFloat, "16777217", "16777216", 0, true},
		// A finite lexical whose magnitude overflows float64 maps to ±INF in the
		// XSD 1.1 value space (ValidateBuiltin accepts it), so it compares equal to
		// the literal INF/-INF spelling rather than being rejected as unparsable.
		{lexicon.TypeFloat, "1e400", lexicon.FloatINF, 0, true},
		{lexicon.TypeFloat, "-1e400", lexicon.FloatNegINF, 0, true},

		// double (the value space is float64, so 16777216 and 16777217 remain
		// distinct — only the float path rounds to single precision).
		{lexicon.TypeDouble, lexicon.FloatINF, lexicon.FloatNegINF, 1, true},
		{lexicon.TypeDouble, "1e10", "9999999999", 1, true},
		{lexicon.TypeDouble, "16777216", "16777217", -1, true},
		{lexicon.TypeDouble, "16777217", "16777216", 1, true},
		// Float64-overflowing double lexicals likewise map to ±INF.
		{lexicon.TypeDouble, "1e400", lexicon.FloatINF, 0, true},
		{lexicon.TypeDouble, "-1e400", lexicon.FloatNegINF, 0, true},

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
		// Year 0000 is a present full XSD 1.1 calendar year, not a missing year
		// component. Mixed-timezone comparisons involving it still use the
		// determinate ±14:00 rule.
		{lexicon.TypeDateTime, "0000-01-10T00:00:00", "0000-01-01T00:00:00Z", 1, true},
		{lexicon.TypeDateTime, "0000-01-01T00:00:00Z", "0000-01-10T00:00:00", -1, true},
		// A leading '+' on the year is not a valid xs:dateTime lexical form, so it
		// must NOT compare equal to the unsigned form (indeterminate, not 0/true).
		{lexicon.TypeDateTime, "+2023-01-01T00:00:00", "2023-01-01T00:00:00", 0, false},
		{lexicon.TypeDateTime, "2023-01-01T00:00:00", "+2023-01-01T00:00:00", 0, false},

		// date
		{lexicon.TypeDate, testDate0, "2023-01-16", -1, true},
		{lexicon.TypeDate, testDate0, testDate0, 0, true},
		{lexicon.TypeDate, "2023-12-31", testJan1, 1, true},
		{lexicon.TypeDate, "2023-01-15Z", "2023-01-15+00:00", 0, true},
		// Huge expanded years exceed an int but must still order correctly
		// (arbitrary-precision year comparison), rather than overflowing to
		// ok=false.
		{lexicon.TypeDate, hugeDate, hugeDatePlus1, -1, true},
		{lexicon.TypeDate, hugeDatePlus1, hugeDate, 1, true},
		{lexicon.TypeDate, hugeDate, hugeDate, 0, true},
		{lexicon.TypeDate, "-" + hugeDate, hugeDate, -1, true},
		// A leading '+' on the year is not a valid xs:date lexical form, so it
		// must NOT compare equal to the unsigned form (indeterminate, not 0/true).
		{lexicon.TypeDate, "+2023-01-01", testJan1, 0, false},
		{lexicon.TypeDate, testJan1, "+2023-01-01", 0, false},

		// time
		{lexicon.TypeTime, testT0, "11:30:00", -1, true},
		{lexicon.TypeTime, testT0, testT0, 0, true},
		{lexicon.TypeTime, "23:59:59", "00:00:00", 1, true},
		{lexicon.TypeTime, "10:30:00Z", "11:30:00+01:00", 0, true},

		// gYear
		{typeGYear, "2023", "2024", -1, true},
		{typeGYear, "2023", "2023", 0, true},
		{typeGYear, "-0001", "2023", -1, true},
		// Huge expanded years compared with arbitrary precision.
		{typeGYear, hugeYear, hugeYearPlus1, -1, true},
		{typeGYear, hugeYearPlus1, hugeYearPlus1, 0, true},
		// A timezone-equivalent huge gYear compares equal (TZ normalization over
		// an arbitrary-precision year).
		{typeGYear, hugeYearPlus1 + "Z", hugeYearPlus1 + "+00:00", 0, true},

		// gYearMonth
		{typeGYearMonth, "2023-01", "2023-02", -1, true},
		{typeGYearMonth, "2023-06", "2023-06", 0, true},

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
		{typeGYearMonth, "2020-06", "2020-06Z", 0, false},
		{typeGMonth, "--06", "--06Z", 0, false},
		{typeGDay, "---15", "---15Z", 0, false},
		{typeGMonthDay, "--06-15", "--06-15Z", 0, false},

		// Mixed-timezone date and time DO get the determinate ±14:00 rule: date
		// has a full calendar date, and compareTime assigns a reference date.
		{lexicon.TypeDate, "2019-12-01", "2020-01-01Z", -1, true}, // interval entirely below bound
		{lexicon.TypeDate, "2020-01-01", "2020-01-01Z", 0, false}, // ±14:00 straddles -> indeterminate
		{lexicon.TypeDate, "0000-01-10", "0000-01-01Z", 1, true},  // year 0000 is present, not missing
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
		{lexicon.TypeDuration, testP1M, "P30D", 0, false}, // indeterminate: months vs days
		{lexicon.TypeDuration, testP1M, "P27D", 1, true},
		{lexicon.TypeDuration, "P27D", testP1M, -1, true},
		{lexicon.TypeDuration, testP1M, "P32D", -1, true},

		// Strict lexical validation gates comparison: the lenient internal date
		// parsers used to accept these, but Compare now validates each operand
		// against the builtin's lexical space first, so malformed input is
		// indeterminate rather than silently comparing equal.
		{typeGYear, "2023abc", "2023", 0, false},                    // trailing junk on a gYear
		{typeGMonthDay, "--02-30", "--02-29", 0, false},             // Feb 30 is not a valid gMonthDay
		{lexicon.TypeDate, "2023-02-29", "2023-02-28", 0, false},    // 2023 is not a leap year
		{lexicon.TypeDate, "2023-01-01+99:99", testDate0, 0, false}, // timezone out of range
		{lexicon.TypeDate, "2023-01-01Zjunk", testDate0, 0, false},  // trailing junk after Z
		// A valid huge-year date with leap Feb 29 (the year is divisible by 4 and
		// not by 100) still compares correctly under strict validation.
		{lexicon.TypeDate, "999999999999999999999996-02-29", "999999999999999999999996-02-28", 1, true},

		// Strict lexical validation now also gates the numeric, float and binary
		// value-comparable types, not just date/time. A non-integer lexical, an
		// out-of-range subtype value, a non-decimal lexical, and the wrong-case
		// "Inf" are all indeterminate rather than silently comparing.
		{lexicon.TypeInteger, "1.0", "1", 0, false},            // not an integer lexical
		{"int", "2147483648", "0", 0, false},                   // out of range for xs:int
		{"unsignedByte", "-1", "0", 0, false},                  // negative is out of range
		{lexicon.TypeDecimal, "1/2", "0.5", 0, false},          // not a decimal lexical
		{lexicon.TypeFloat, "Inf", lexicon.FloatINF, 0, false}, // "Inf" is not a valid float lexical
		// NBSP-padded values are not trimmed by XSD whitespace handling, so they
		// stay invalid and compare indeterminate.
		{lexicon.TypeInteger, " 1 ", "1", 0, false},

		// xs:dateTime/xs:time 24:00:00 is the valid end-of-day form; it lives in
		// the same value space as 00:00:00 of the next day and must compare equal.
		{lexicon.TypeDateTime, "2024-01-01T24:00:00", "2024-01-02T00:00:00", 0, true},
		{lexicon.TypeTime, "24:00:00", "00:00:00", 0, true},

		// Date/time lexicals are whiteSpace=collapse, so XSD-whitespace padding
		// (space/tab/CR/LF) must be stripped before validation and parsing, exactly
		// like the numeric/binary paths. A padded operand therefore compares equal
		// to its trimmed form rather than being rejected.
		{lexicon.TypeDate, " 2023-01-01 ", testJan1, 0, true},
		{lexicon.TypeDateTime, "\t2023-01-15T10:30:00\n", testDT0, 0, true},
		{lexicon.TypeTime, " 10:30:00 ", testT0, 0, true},
		{typeGYear, " 2023 ", "2023", 0, true},
		// String-family and unrecognized builtins have no numeric value-space
		// comparison defined: Compare must NOT route them through CompareDecimal,
		// so "5.0" vs "5" for xs:string (and an unknown builtin) is indeterminate.
		{lexicon.TypeString, "5.0", "5", 0, false},
		{"definitely-not-a-builtin", "5.0", "5", 0, false},

		// Huge duration components must not overflow: months are parsed as big.Int
		// and seconds as big.Rat, so a year/day count far beyond int64 range that
		// passes ValidateBuiltin also compares correctly rather than failing to
		// parse.
		{lexicon.TypeDuration, "P999999999999999999999999Y", "P999999999999999999999999Y", 0, true},
		{lexicon.TypeDuration, "P999999999999999999999998Y", "P999999999999999999999999Y", -1, true},
		{lexicon.TypeDuration, "P99999999999999999999D", "PT8639999999999999999913600S", 0, true},

		// NBSP is not XSD whitespace, so a NBSP-padded date/time operand stays
		// lexically invalid and compares indeterminate.
		{lexicon.TypeDate, " 2023-01-01", testJan1, 0, false},
		{lexicon.TypeTime, "10:30:00 ", testT0, 0, false},
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

// TestNormalize verifies that Normalize applies the XSD whiteSpace facet using
// only the four ASCII XSD whitespace characters (#x20, #x9, #xD, #xA). Unicode
// whitespace such as NBSP (U+00A0) must NOT be treated as whitespace, so an
// invalid value containing it survives normalization and is later rejected by
// lexical validation.
func TestNormalize(t *testing.T) {
	const nbsp = " "
	tests := []struct {
		typ  string
		in   string
		want string
	}{
		// preserve: xs:string leaves everything untouched.
		{lexicon.TypeString, "  a\tb  ", "  a\tb  "},
		// replace: xs:normalizedString maps the ASCII whitespace to spaces but
		// does not collapse or trim, and leaves NBSP alone.
		{"normalizedString", "a\tb\nc\rd", "a b c d"},
		{"normalizedString", "a" + nbsp + "b", "a" + nbsp + "b"},
		// collapse (the default for token and other types): ASCII whitespace runs
		// collapse to a single space and the ends trim.
		{lexicon.TypeToken, "  a\t b  ", "a b"},
		{lexicon.TypeToken, "a  b", "a b"},
		// collapse must NOT touch NBSP: it is preserved verbatim so the value
		// remains lexically invalid afterwards.
		{lexicon.TypeToken, "a" + nbsp + "b", "a" + nbsp + "b"},
		{lexicon.TypeInteger, nbsp + "5", nbsp + "5"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%q", tt.typ, tt.in), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, value.Normalize(tt.in, tt.typ))
		})
	}
}

// TestIsFloatNaN verifies that only the bare "NaN" lexical form denotes NaN: the
// sign-prefixed "+NaN"/"-NaN" are not valid XSD xs:float/xs:double lexical forms,
// and INF and finite values are not NaN.
func TestIsFloatNaN(t *testing.T) {
	// Only the bare "NaN" denotes NaN. The sign-prefixed +NaN/-NaN are not valid
	// XSD lexical forms and must be rejected, so a facet value="+NaN" cannot
	// value-match an instance "NaN".
	require.True(t, value.IsFloatNaN(lexicon.FloatNaN), "bare NaN is NaN")
	require.False(t, value.IsFloatNaN("+NaN"), "+NaN is not a valid NaN form")
	require.False(t, value.IsFloatNaN("-NaN"), "-NaN is not a valid NaN form")
	// strconv.ParseFloat accepts these lenient spellings, but they are not valid
	// XSD lexical forms for NaN, so IsFloatNaN must reject them.
	require.False(t, value.IsFloatNaN("nan"), "lowercase nan is not a valid NaN form")
	require.False(t, value.IsFloatNaN("NAN"), "uppercase NAN is not a valid NaN form")
	require.False(t, value.IsFloatNaN("Infinity"), "Infinity is not a valid NaN form")
	require.False(t, value.IsFloatNaN(lexicon.FloatINF), "INF is not NaN")
	require.False(t, value.IsFloatNaN("1.5"), "finite value is not NaN")
}

// TestTimezoneUppercaseZOnly verifies that the timezone designator is the
// uppercase 'Z' only. XSD permits "Z" but never the lowercase "z"; the latter
// must fail lexical validation and must NOT compare or canonicalize as equal to
// the uppercase form.
func TestTimezoneUppercaseZOnly(t *testing.T) {
	t.Parallel()

	// Uppercase Z is a valid designator across the date/time family; lowercase z
	// is rejected for every one of them.
	zTypes := []struct {
		typ      string
		validZ   string
		invalidz string
	}{
		{"date", "2023-01-01Z", "2023-01-01z"},
		{lexicon.TypeDateTime, "2023-01-01T10:30:00Z", "2023-01-01T10:30:00z"},
		{lexicon.TypeTime, "10:30:00Z", "10:30:00z"},
		{typeGYear, "2023Z", "2023z"},
		{typeGYearMonth, "2023-01Z", "2023-01z"},
		{typeGMonth, "--06Z", "--06z"},
		{typeGDay, "---15Z", "---15z"},
		{typeGMonthDay, "--06-01Z", "--06-01z"},
	}
	for _, tt := range zTypes {
		t.Run(tt.typ, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, value.ValidateBuiltin(tt.validZ, tt.typ, value.Version11), "uppercase Z must be valid")
			require.Error(t, value.ValidateBuiltin(tt.invalidz, tt.typ, value.Version11), "lowercase z must be invalid")
		})
	}

	// Compare must treat a lowercase-z lexical as indeterminate (ok=false), not as
	// equal to the uppercase-Z form: validation runs first and rejects "z".
	_, ok := value.Compare("2023-01-01z", "2023-01-01Z", "date")
	require.False(t, ok, "lowercase z must not compare as a valid date")

	// CanonicalKey must reject the lowercase-z lexical (ok=false) so it never keys
	// equal to the uppercase-Z form on the IDC path.
	keyZ, okZ := value.CanonicalKey("2023-01-01Z", "date")
	require.True(t, okZ, "uppercase Z must canonicalize")
	_, okz := value.CanonicalKey("2023-01-01z", "date")
	require.False(t, okz, "lowercase z must not canonicalize")
	require.NotEmpty(t, keyZ)
}

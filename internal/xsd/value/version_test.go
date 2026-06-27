package value_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/stretchr/testify/require"
)

// TestValidateBuiltinVersion exercises the XSD 1.0-vs-1.1 lexical differences
// gated by ValidateBuiltin's version argument: the "+INF" float/double form and
// the year "0000" on the date types are valid only under XSD 1.1.
func TestValidateBuiltinVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		typeName string
		valid10  bool
		valid11  bool
	}{
		// +INF: 1.1-only.
		{"+INF float", "+INF", lexicon.TypeFloat, false, true},
		{"+INF double", "+INF", lexicon.TypeDouble, false, true},
		// INF / -INF / NaN: valid in both versions.
		{"INF float", "INF", lexicon.TypeFloat, true, true},
		{"-INF float", "-INF", lexicon.TypeFloat, true, true},
		{"NaN float", "NaN", lexicon.TypeFloat, true, true},
		// Ordinary numeric forms: valid in both (sign on numeric is unchanged).
		{"+3.14 float", "+3.14", lexicon.TypeFloat, true, true},
		{"-1.5 double", "-1.5", lexicon.TypeDouble, true, true},
		// year 0000: 1.1-only across the date types.
		{"date 0000", "0000-01-01", lexicon.TypeDate, false, true},
		{"dateTime 0000", "0000-01-01T00:00:00", lexicon.TypeDateTime, false, true},
		{"gYear 0000", "0000", lexicon.TypeGYear, false, true},
		{"gYearMonth 0000", "0000-01", lexicon.TypeGYearMonth, false, true},
		// Ordinary and negative (BC) years: valid in both.
		{"date 2020", "2020-01-01", lexicon.TypeDate, true, true},
		{"date -0001", "-0001-01-01", lexicon.TypeDate, true, true},
		{"gYear 2020", "2020", lexicon.TypeGYear, true, true},
		{"gYear -0001", "-0001", lexicon.TypeGYear, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err10 := value.ValidateBuiltin(tt.value, tt.typeName, value.Version10)
			if tt.valid10 {
				require.NoError(t, err10, "XSD 1.0 should accept %q for %s", tt.value, tt.typeName)
			} else {
				require.Error(t, err10, "XSD 1.0 should reject %q for %s", tt.value, tt.typeName)
			}
			err11 := value.ValidateBuiltin(tt.value, tt.typeName, value.Version11)
			if tt.valid11 {
				require.NoError(t, err11, "XSD 1.1 should accept %q for %s", tt.value, tt.typeName)
			} else {
				require.Error(t, err11, "XSD 1.1 should reject %q for %s", tt.value, tt.typeName)
			}
		})
	}
}

// TestValidateBuiltin11Types covers the lexical spaces of the XSD 1.1-only
// built-in datatypes. These are pure lexical checks at the value layer; the xsd
// registry gates their availability by version. All cases use non-zero years, so
// the version argument is immaterial here.
func TestValidateBuiltin11Types(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		typeName string
		valid    bool
	}{
		// xs:dateTimeStamp: xs:dateTime with a REQUIRED timezone.
		{"dateTimeStamp with Z", "2020-01-01T00:00:00Z", lexicon.TypeDateTimeStamp, true},
		{"dateTimeStamp with offset", "2020-01-01T00:00:00+05:00", lexicon.TypeDateTimeStamp, true},
		{"dateTimeStamp without tz", "2020-01-01T00:00:00", lexicon.TypeDateTimeStamp, false},
		// xs:dayTimeDuration: only D/H/M/S components.
		{"dayTimeDuration P1DT2H", "P1DT2H", lexicon.TypeDayTimeDuration, true},
		{"dayTimeDuration PT5M", "PT5M", lexicon.TypeDayTimeDuration, true},
		{"dayTimeDuration with year", "P1Y", lexicon.TypeDayTimeDuration, false},
		{"dayTimeDuration empty", "P", lexicon.TypeDayTimeDuration, false},
		// xs:yearMonthDuration: only Y/M components.
		{"yearMonthDuration P1Y2M", "P1Y2M", lexicon.TypeYearMonthDuration, true},
		{"yearMonthDuration P3M", "P3M", lexicon.TypeYearMonthDuration, true},
		{"yearMonthDuration with day", "P1D", lexicon.TypeYearMonthDuration, false},
		{"yearMonthDuration empty", "P", lexicon.TypeYearMonthDuration, false},
		// xs:anyAtomicType: abstract, no direct lexical constraint.
		{"anyAtomicType anything", "anything", lexicon.TypeAnyAtomicType, true},
		// xs:error: empty value space — nothing is valid.
		{"error empty string", "", lexicon.TypeError, false},
		{"error any value", "x", lexicon.TypeError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := value.ValidateBuiltin(tt.value, tt.typeName, value.Version11)
			if tt.valid {
				require.NoError(t, err, "%s should accept %q", tt.typeName, tt.value)
			} else {
				require.Error(t, err, "%s should reject %q", tt.typeName, tt.value)
			}
		})
	}
}

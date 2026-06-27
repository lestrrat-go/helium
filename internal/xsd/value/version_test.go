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

package xpath

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNumberToString(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want string
	}{
		// Special values.
		{"NaN", math.NaN(), "NaN"},
		{"positive infinity", math.Inf(1), "Infinity"},
		{"negative infinity", math.Inf(-1), "-Infinity"},
		{"positive zero", 0.0, "0"},
		{"negative zero", math.Copysign(0, -1), "0"},

		// Integers within int32 range (libxml2 uses %d).
		{"integer 1", 1.0, "1"},
		{"integer -1", -1.0, "-1"},
		{"integer 42", 42.0, "42"},
		{"integer 1000", 1000.0, "1000"},
		// MaxInt32 itself is excluded by strict < comparison (matching libxml2).
		{"integer max int32", float64(math.MaxInt32), "2.147483647e+09"},
		{"integer near max", float64(math.MaxInt32 - 1), "2147483646"},

		// Simple decimals.
		{"decimal 3.14", 3.14, "3.14"},
		{"decimal 0.5", 0.5, "0.5"},
		{"decimal -2.5", -2.5, "-2.5"},
		{"decimal 0.1", 0.1, "0.1"},
		{"decimal 0.0001", 0.0001, "0.0001"},

		// Boundary: libxml2 uses fixed for abs >= 1e-5 (LOWER_DOUBLE).
		{"1e-5 fixed", 0.00001, "0.00001"},
		{"5e-5 fixed", 0.00005, "0.00005"},
		{"-1e-5 fixed", -0.00001, "-0.00001"},

		// Below LOWER_DOUBLE: scientific notation.
		{"1e-6 scientific", 0.000001, "1e-06"},
		{"1e-10 scientific", 1e-10, "1e-10"},

		// At/above UPPER_DOUBLE (1e9): scientific for non-integers.
		// Note: 1.5e9 is actually an integer (1500000000) within int32 range.
		{"1.5e9 integer", 1.5e9, "1500000000"},
		{"1e10 scientific", 1e10, "1e+10"},
		{"1.23e12 scientific", 1.23e12, "1.23e+12"},
		{"non-integer above 1e9", 1.0000000005e9, "1.0000000005e+09"},

		// Large integers within int32 range still use integer format.
		{"1e9 integer", 1e9, "1000000000"},
		{"2e9 integer", 2e9, "2000000000"},

		// Integer just beyond int32 range: scientific.
		{"3e9 scientific", 3e9, "3e+09"},

		// Trailing zero stripping in fixed notation.
		{"trailing zeros fixed", 1.50, "1.5"},
		{"trailing zeros fixed 2", 1.500000, "1.5"},
		{"trailing zeros fixed 3", 100.0, "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := numberToString(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

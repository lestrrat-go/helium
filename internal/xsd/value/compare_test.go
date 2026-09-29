package value_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/stretchr/testify/require"
)

// ratCompareDecimal is the reference result for CompareDecimal: the ordering of
// the two operands as parsed by math/big.Rat, or -2 when either fails to parse.
func ratCompareDecimal(a, b string) int {
	ra, ok1 := new(big.Rat).SetString(a)
	rb, ok2 := new(big.Rat).SetString(b)
	if !ok1 || !ok2 {
		return -2
	}
	return ra.Cmp(rb)
}

// decimalCorpus builds operands covering every shape of the xs:decimal lexical
// grammar (signs, leading/trailing zeros, ".5" and "5." forms, zeros, 40-digit
// parts) plus forms outside that grammar which big.Rat still parses or rejects.
func decimalCorpus() []string {
	signs := []string{"", "+", "-"}
	long := strings.Repeat("9", 20) + strings.Repeat("0", 19) + "1"
	ints := []string{"", "0", "00", "5", "005", "10", "123", "0123", long, "0" + long}
	fracs := []string{"", ".", ".0", ".000", ".5", ".50", ".05", ".1", ".123", "." + long, "." + long + "000"}

	var out []string
	for _, s := range signs {
		for _, i := range ints {
			for _, f := range fracs {
				out = append(out, s+i+f)
			}
		}
	}
	return append(out,
		// Outside the fast-path grammar: big.Rat parses these.
		"1e5", "1E-3", "-2.5e+1", "1/2", "-3/6", "0x10", "0b101", "0o17", "0x1p-2", "0x_1", "1e400",
		// Outside the grammar and rejected by big.Rat.
		"", " ", " 1", "1 ", "\t1", "1.2.3", "abc", "1_0", "--1", "+-1", "1-", "1/0", "1/", "/2",
		"NaN", "INF", "-INF", "1..", "..1", ".e1", "٣", "1 ",
	)
}

func TestCompareDecimal(t *testing.T) {
	t.Run("explicit cases", func(t *testing.T) {
		cases := []struct {
			a, b string
			want int
		}{
			{"1", "2", -1},
			{"2", "1", 1},
			{"1.0", "1", 0},
			{"-0", "0", 0},
			{"-0.000", "+.0", 0},
			{"0.", ".0", 0},
			{"5.", "5", 0},
			{".5", "0.5", 0},
			{"-.5", "-0.50", 0},
			{"-1", "-2", 1},
			{"-10", "-9", -1},
			{"0.05", "0.5", -1},
			{"0.5", "0.51", -1},
			{"007", "7.000", 0},
			{"99999999999999999999999999999999999999.1", "99999999999999999999999999999999999999.09", 1},
			{"1e5", "100000", 0},
			{"1/2", "0.5", 0},
			{"0x10", "16", 0},
			{"abc", "1", -2},
			{"1", "", -2},
			{"1.2.3", "1", -2},
		}
		for _, tc := range cases {
			require.Equal(t, tc.want, value.CompareDecimal(tc.a, tc.b), "CompareDecimal(%q, %q)", tc.a, tc.b)
			require.Equal(t, tc.want, ratCompareDecimal(tc.a, tc.b), "reference(%q, %q)", tc.a, tc.b)
		}
	})

	t.Run("matches big.Rat on every corpus pair", func(t *testing.T) {
		corpus := decimalCorpus()
		for _, a := range corpus {
			for _, b := range corpus {
				require.Equal(t, ratCompareDecimal(a, b), value.CompareDecimal(a, b), "CompareDecimal(%q, %q)", a, b)
			}
		}
	})
}

func FuzzCompareDecimal(f *testing.F) {
	corpus := decimalCorpus()
	for i, a := range corpus {
		f.Add(a, corpus[(i*7+3)%len(corpus)])
	}
	f.Fuzz(func(t *testing.T, a, b string) {
		require.Equal(t, ratCompareDecimal(a, b), value.CompareDecimal(a, b), "CompareDecimal(%q, %q)", a, b)
	})
}

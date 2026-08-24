package xslt3

import "testing"

func TestFrenchCardinalTens(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{40, "quarante"},
		{41, "quarante et un"},
		{42, "quarante-deux"},
		{45, "quarante-cinq"},
		{49, "quarante-neuf"},
		{71, "soixante et onze"},
		{80, "quatre-vingts"},
		{91, "quatre-vingt-onze"},
	}
	for _, tc := range cases {
		if got := frenchCardinal(tc.n); got != tc.want {
			t.Errorf("frenchCardinal(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

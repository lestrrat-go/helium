package xslt3

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatBigNumberListWordTokens(t *testing.T) {
	testCases := []struct {
		name    string
		number  int64
		ordinal string
		want    string
	}{
		{
			name:   "largest signed 32-bit integer",
			number: 2_147_483_647,
			want: "two billion one hundred and forty seven million four hundred and eighty three thousand " +
				"six hundred and forty seven",
		},
		{
			name:   "first integer above signed 32-bit range",
			number: 2_147_483_648,
			want: "two billion one hundred and forty seven million four hundred and eighty three thousand " +
				"six hundred and forty eight",
		},
		{
			name:   "largest cardinal below trillion fallback",
			number: numberTrillion - 1,
			want: "nine hundred and ninety nine billion nine hundred and ninety nine million " +
				"nine hundred and ninety nine thousand nine hundred and ninety nine",
		},
		{
			name:   "cardinal at trillion fallback",
			number: numberTrillion,
			want:   "1000000000000",
		},
		{
			name:    "ordinal at trillion",
			number:  numberTrillion,
			ordinal: "yes",
			want:    "one trillionth",
		},
		{
			name:    "ordinal above trillion",
			number:  numberTrillion + 1,
			ordinal: "yes",
			want:    "one trillion first",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := formatBigNumberList(
				[]*big.Int{big.NewInt(testCase.number)},
				"w",
				"",
				0,
				"en",
				testCase.ordinal,
			)
			require.Equal(t, testCase.want, got)
		})
	}
}

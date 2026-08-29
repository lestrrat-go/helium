package xslt3

import (
	"math/big"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const ordinalYes = "yes"

func TestExecNumberStartAt(t *testing.T) {
	testCases := []struct {
		name    string
		startAt string
		format  string
		ordinal string
		want    string
	}{
		{
			name:    "static integer above signed 32-bit range",
			startAt: "2147483648",
			want: "two billion one hundred and forty seven million four hundred and eighty three thousand " +
				"six hundred and forty eight",
		},
		{
			name:    "dynamic integer above signed 32-bit range",
			startAt: "{2147483648}",
			want: "two billion one hundred and forty seven million four hundred and eighty three thousand " +
				"six hundred and forty eight",
		},
		{
			name:    "minimum signed 64-bit integer in words",
			startAt: "-9223372036854775808",
			want:    "minus 9223372036854775808",
		},
		{
			name:    "minimum signed 64-bit ordinal in words",
			startAt: "-9223372036854775808",
			ordinal: ordinalYes,
			want:    "minus 9223372036854775808th",
		},
		{
			name:    "alphabetic above signed 32-bit range",
			startAt: "2147483648",
			format:  "a",
			want:    "fxshrxx",
		},
		{
			name:    "numeric ordinal above signed 32-bit range",
			startAt: "2147483648",
			format:  "1",
			ordinal: ordinalYes,
			want:    "2147483648th",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			format := testCase.format
			if format == "" {
				format = "w"
			}
			ordinal := ""
			if testCase.ordinal != "" {
				ordinal = ` ordinal="` + testCase.ordinal + `"`
			}
			stylesheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output method="text"/>
<xsl:template match="/"><xsl:number value="1" start-at="` + testCase.startAt + `" format="` + format + `"` + ordinal + `/></xsl:template>
</xsl:stylesheet>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet))
			require.NoError(t, err)
			ss, err := NewCompiler().Compile(t.Context(), doc)
			require.NoError(t, err)

			source, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)
			got, err := ss.Transform(source).Serialize(t.Context())
			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}

	t.Run("invalid dynamic value raises XTDE0030", func(t *testing.T) {
		stylesheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output method="text"/>
<xsl:template match="/"><xsl:number value="1" start-at="{'invalid'}"/></xsl:template>
</xsl:stylesheet>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet))
		require.NoError(t, err)
		ss, err := NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)
		_, err = ss.Transform(source).Serialize(t.Context())
		require.ErrorContains(t, err, errCodeXTDE0030)
	})
}

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
			ordinal: ordinalYes,
			want:    "one trillionth",
		},
		{
			name:    "ordinal above trillion",
			number:  numberTrillion + 1,
			ordinal: ordinalYes,
			want:    "one trillion first",
		},
		{
			name:   "minimum signed 64-bit integer",
			number: minInt64,
			want:   "minus 9223372036854775808",
		},
		{
			name:    "minimum signed 64-bit ordinal",
			number:  minInt64,
			ordinal: ordinalYes,
			want:    "minus 9223372036854775808th",
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

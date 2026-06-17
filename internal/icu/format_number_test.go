package icu_test

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/icu"
	"github.com/stretchr/testify/require"
)

// TestFormatNumberHugeFractionalPicture guards against an overflow in the
// float formatting path: a picture with hundreds of fractional digit
// placeholders made math.Pow(10, MaxFracDigits) overflow to +Inf, which
// turned a finite value into NaN and panicked big.Float.SetFloat64(NaN).
// The result must stay finite (no panic) and remain a well-formed decimal.
func TestFormatNumberHugeFractionalPicture(t *testing.T) {
	const fracDigits = 400
	picture := "0." + strings.Repeat("0", fracDigits)

	out, err := icu.FormatNumber(1.2, false, false, false, false, nil, picture, icu.DefaultDecimalFormat())
	require.NoError(t, err)

	// Expect "<int>.<fracDigits digits>": a finite value, not "NaN"/"Infinity".
	intPart, fracPart, ok := strings.Cut(out, ".")
	require.True(t, ok, "expected a decimal point in %q", out)
	require.Equal(t, "1", intPart, "integer part of %q", out)
	require.Equal(t, fracDigits, len(fracPart), "fractional digit count of %q", out)
	require.True(t, strings.HasPrefix(fracPart, "1"), "fractional part of %q", out)
	for _, r := range out {
		require.True(t, r == '.' || (r >= '0' && r <= '9'), "unexpected char %q in %q", r, out)
	}
}

// TestFormatNumberHugeFractionalPicturePrecise exercises the same huge picture
// on the precise (*big.Rat) path used by real xpath3/xslt3 callers. It must not
// overflow and must produce the exact value padded with trailing zeros.
func TestFormatNumberHugeFractionalPicturePrecise(t *testing.T) {
	const fracDigits = 400
	picture := "0." + strings.Repeat("0", fracDigits)

	// 1.25 is exactly representable in decimal.
	precise := new(big.Rat).SetFrac64(5, 4)
	out, err := icu.FormatNumber(1.25, false, false, false, false, precise, picture, icu.DefaultDecimalFormat())
	require.NoError(t, err)
	require.Equal(t, "1.25"+strings.Repeat("0", fracDigits-2), out)
}

// TestFormatNumberFloatTinyValuePreserved guards against capping fractional
// precision: a tiny finite value with enough fractional placeholders to hold
// its leading significant digit must keep that digit (not round to zero).
func TestFormatNumberFloatTinyValuePreserved(t *testing.T) {
	picture := "0." + strings.Repeat("0", 20)
	out, err := icu.FormatNumber(1e-20, false, false, false, false, nil, picture, icu.DefaultDecimalFormat())
	require.NoError(t, err)
	require.Equal(t, "0."+strings.Repeat("0", 19)+"1", out)
}

// TestFormatNumberFloatLargeValueFinite guards against f*scale overflowing to
// +Inf for large finite inputs, which previously produced "<nil>." output. The
// integer part must format as digits, never "<nil>".
func TestFormatNumberFloatLargeValueFinite(t *testing.T) {
	out, err := icu.FormatNumber(math.MaxFloat64, false, false, false, false, nil, "0.0", icu.DefaultDecimalFormat())
	require.NoError(t, err)
	require.NotContains(t, out, "<nil>", "got %q", out)
	intPart, _, ok := strings.Cut(out, ".")
	require.True(t, ok, "got %q", out)
	require.NotEmpty(t, intPart)
	for _, r := range intPart {
		require.True(t, r >= '0' && r <= '9', "non-digit %q in integer part of %q", r, out)
	}
}

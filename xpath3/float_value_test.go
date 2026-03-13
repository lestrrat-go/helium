package xpath3_test

import (
	"math"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestNewFloat_SpecialValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   float64
		isNaN   bool
		isPosI  bool
		isNegI  bool
		isNegZ  bool
		isZero  bool
		signbit bool
	}{
		{"NaN", math.NaN(), true, false, false, false, false, false},
		{"+Inf", math.Inf(1), false, true, false, false, false, false},
		{"-Inf", math.Inf(-1), false, false, true, false, false, true},
		{"-0", math.Copysign(0, -1), false, false, false, true, true, true},
		{"+0", 0, false, false, false, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fv := xpath3.NewFloat(tt.input)
			require.Equal(t, tt.isNaN, fv.IsNaN(), "IsNaN")
			require.Equal(t, tt.isPosI, fv.IsInf(1), "IsInf(+1)")
			require.Equal(t, tt.isNegI, fv.IsInf(-1), "IsInf(-1)")
			require.Equal(t, tt.isZero, fv.IsZero(), "IsZero")
			require.Equal(t, tt.signbit, fv.Signbit(), "Signbit")
		})
	}
}

func TestNewFloat_Float32Overflow(t *testing.T) {
	t.Parallel()

	// Values larger than float32 max should overflow to ±Inf
	fv := xpath3.NewFloat(math.MaxFloat64)
	require.True(t, fv.IsInf(1), "MaxFloat64 should overflow to +Inf for xs:float")

	fv = xpath3.NewFloat(-math.MaxFloat64)
	require.True(t, fv.IsInf(-1), "negative MaxFloat64 should overflow to -Inf for xs:float")

	// float32 subnormal underflow: values smaller than float32 min subnormal → ±0
	tiny := math.SmallestNonzeroFloat64
	fv = xpath3.NewFloat(tiny)
	require.True(t, fv.IsZero(), "SmallestNonzeroFloat64 should underflow to 0 for xs:float")
}

func TestNewDouble_PreservesRange(t *testing.T) {
	t.Parallel()

	// MaxFloat64 should remain finite as a double
	fv := xpath3.NewDouble(math.MaxFloat64)
	require.False(t, fv.IsInf(0), "MaxFloat64 should stay finite for xs:double")
	require.Equal(t, math.MaxFloat64, fv.Float64())
}

func TestFloatValue_Neg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  float64
		expect float64
	}{
		{"+1 → -1", 1.0, -1.0},
		{"-1 → +1", -1.0, 1.0},
		{"+0 → -0", 0.0, math.Copysign(0, -1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fv := xpath3.NewDouble(tt.input)
			neg := fv.Neg()
			require.Equal(t, tt.expect, neg.Float64())
			require.Equal(t, math.Signbit(tt.expect), neg.Signbit())
		})
	}

	// -0 → +0
	t.Run("-0 → +0", func(t *testing.T) {
		t.Parallel()
		fv := xpath3.NewDouble(math.Copysign(0, -1))
		neg := fv.Neg()
		require.True(t, neg.IsZero())
		require.False(t, neg.Signbit())
	})

	// NaN negation stays NaN
	t.Run("NaN → NaN", func(t *testing.T) {
		t.Parallel()
		fv := xpath3.NewDouble(math.NaN())
		require.True(t, fv.Neg().IsNaN())
	})

	// +Inf → -Inf
	t.Run("+Inf → -Inf", func(t *testing.T) {
		t.Parallel()
		fv := xpath3.NewDouble(math.Inf(1))
		require.True(t, fv.Neg().IsInf(-1))
	})
}

func TestFloatValue_WithPrecision(t *testing.T) {
	t.Parallel()

	// Promote float to double
	fv := xpath3.NewFloat(1.5)
	dbl := fv.WithPrecision(xpath3.PrecisionDouble)
	require.Equal(t, uint(xpath3.PrecisionDouble), dbl.Precision())
	require.Equal(t, 1.5, dbl.Float64())

	// Special values preserve through precision change
	fv = xpath3.NewFloat(math.NaN())
	require.True(t, fv.WithPrecision(xpath3.PrecisionDouble).IsNaN())
}

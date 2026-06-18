package xpath3_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestDurationFractionalSeconds guards against double-counting fractional
// seconds when a dayTimeDuration carries an exact FracSec component alongside
// the float64 Seconds total.
func TestDurationFractionalSeconds(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "div fractional by whole",
			expr: `xs:dayTimeDuration("PT1.5S") div xs:dayTimeDuration("PT1S")`,
			want: "1.5",
		},
		{
			name: "div fractional by fractional",
			expr: `xs:dayTimeDuration("PT1.5S") div xs:dayTimeDuration("PT0.5S")`,
			want: "3",
		},
		{
			// Fraction so close to 1 that float64 Seconds rounds up to the next
			// integer; the whole-second count must still be 0, not 1.
			name: "div near-one fraction by whole",
			expr: `xs:dayTimeDuration("PT0.999999999999999999999999999999S") div xs:dayTimeDuration("PT1S")`,
			want: "0.999999999999999999999999999999",
		},
		{
			name: "div equal fractional durations",
			expr: `xs:dayTimeDuration("PT2.25S") div xs:dayTimeDuration("PT2.25S")`,
			want: "1",
		},
		{
			name: "add fractional durations",
			expr: `xs:dayTimeDuration("PT1.5S") + xs:dayTimeDuration("PT1S")`,
			want: "PT2.5S",
		},
		{
			name: "subtract fractional durations",
			expr: `xs:dayTimeDuration("PT2.5S") - xs:dayTimeDuration("PT1S")`,
			want: "PT1.5S",
		},
		{
			name: "multiply fractional duration",
			expr: `xs:dayTimeDuration("PT1.5S") * 2`,
			want: "PT3S",
		},
		{
			name: "compare fractional durations equal",
			expr: `xs:dayTimeDuration("PT1.5S") eq xs:dayTimeDuration("PT1.5S")`,
			want: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := evalExpr(t, doc, tt.expr)
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			got, err := xpath3.AtomicToString(av)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestDurationExactArithmeticFormatting guards against the regression where
// exact-rational duration arithmetic stringified to an invalid lexical form
// because the whole-second part was taken from a rounded float64 while a
// separate exact fraction was still emitted (producing e.g. "PT1.1.S").
func TestDurationExactArithmeticFormatting(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			// Near-carry fraction: PT0.333...S * 3 = PT0.999...S, whose float64
			// total rounds UP to 1.0. The whole part must stay 0 and the fraction
			// be emitted exactly — never "PT1.1.S".
			name: "fraction times three carries near a whole second",
			expr: `xs:dayTimeDuration("PT0.333333333333333333333333333333S") * 3`,
			want: "PT0.999999999999999999999999999999S",
		},
		{
			// A value just below a whole second: float64 Seconds rounds up to 1.0
			// but the exact rational keeps the integer part at 0.
			name: "value just below a whole second",
			expr: `xs:dayTimeDuration("PT0.9999999999999999999S") * 1`,
			want: "PT0.9999999999999999999S",
		},
		{
			// A long EXACT terminating fraction (50 nines, denominator 10^50) must
			// be rendered in full, NOT rounded up to a whole second and stripped to
			// "PT0S". Guards the exactFractionDigits 40-digit-cap regression.
			name: "long exact terminating fraction is not capped",
			expr: `xs:dayTimeDuration("PT0.99999999999999999999999999999999999999999999999999S") * 1`,
			want: "PT0.99999999999999999999999999999999999999999999999999S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := evalExpr(t, doc, tt.expr)
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			got, err := xpath3.AtomicToString(av)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestDurationFractionFormattingPrecision verifies exactFractionDigits renders a
// long EXACT terminating fraction in full while a true NON-terminating fraction
// is capped (the only case that legitimately rounds).
func TestDurationFractionFormattingPrecision(t *testing.T) {
	tests := []struct {
		name   string
		secRat *big.Rat
		want   string
	}{
		{
			// 1/3 of a second is non-terminating; capped at 40 fractional digits.
			name:   "non-terminating fraction capped at 40 digits",
			secRat: big.NewRat(1, 3),
			want:   "PT0.3333333333333333333333333333333333333333S",
		},
		{
			// (10^60 - 1)/10^60 is exactly representable (60 nines); rendered in full.
			name: "long exact terminating fraction rendered in full",
			secRat: func() *big.Rat {
				den := new(big.Int).Exp(big.NewInt(10), big.NewInt(60), nil)
				num := new(big.Int).Sub(den, big.NewInt(1))
				return new(big.Rat).SetFrac(num, den)
			}(),
			want: "PT0." + strings.Repeat("9", 60) + "S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			av := xpath3.AtomicValue{
				TypeName: xpath3.TypeDayTimeDuration,
				Value:    xpath3.Duration{SecRat: tt.secRat},
			}
			got, err := xpath3.AtomicToString(av)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaDerivedDurationArithmetic verifies that a schema-derived duration
// value (custom TypeName whose BaseType is a built-in duration) participates in
// arithmetic by promoting via BaseType, and that the exact result canonicalizes
// as a map key equal to the parsed built-in equivalent.
func TestSchemaDerivedDurationArithmetic(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	parsed, err := xpath3.CastFromString("PT11S", xpath3.TypeDayTimeDuration)
	require.NoError(t, err)
	derived := xpath3.AtomicValue{
		TypeName: "Q{urn:test}myDTD",
		BaseType: xpath3.TypeDayTimeDuration,
		Value:    parsed.Value,
	}

	vars := xpath3.NewVariables()
	vars.Set("d", xpath3.SingleAtomic(derived))

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(vars)

	// PT11S * 0.1 must succeed (no XPTY0004) and yield PT1.1S exactly.
	seq := evalExprWithEval(t, eval, doc, `$d * 0.1`)
	require.Equal(t, 1, seq.Len())
	av := seq.Get(0).(xpath3.AtomicValue)
	got, err := xpath3.AtomicToString(av)
	require.NoError(t, err)
	require.Equal(t, "PT1.1S", got)

	// The arithmetic result must canonicalize as a map key equal to a parsed
	// xs:dayTimeDuration("PT1.1S").
	parsedResult, err := xpath3.CastFromString("PT1.1S", xpath3.TypeDayTimeDuration)
	require.NoError(t, err)
	m := xpath3.NewMap([]xpath3.MapEntry{
		{Key: parsedResult, Value: xpath3.SingleInteger(11)},
	})
	v, ok := m.Get(av)
	require.True(t, ok)
	require.Equal(t, int64(11), v.Get(0).(xpath3.AtomicValue).IntegerVal())
}

// TestDurationWholeSecondOverflow guards against the regression where formatting
// an exact whole-second magnitude above math.MaxInt64 wrapped through an int64
// conversion and emitted malformed negative components (e.g.
// "P-106751991167300DT-15H-30M-8S").
func TestDurationWholeSecondOverflow(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// 9223372036854775808 = math.MaxInt64 + 1 whole seconds.
			name:  "whole seconds just above int64 max",
			input: "PT9223372036854775808S",
			want:  "P106751991167300DT15H30M8S",
		},
		{
			// A far larger whole-second total still decomposes correctly in big.Int.
			name:  "whole seconds far above int64 max",
			input: "PT100000000000000000000S",
			want:  "P1157407407407407DT9H46M40S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			av, err := xpath3.CastFromString(tt.input, xpath3.TypeDayTimeDuration)
			require.NoError(t, err)
			got, err := xpath3.AtomicToString(av)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaDerivedDurationAggregate verifies that fn:sum and fn:avg classify a
// schema-derived duration (custom TypeName whose BaseType is a built-in
// duration) via its BaseType rather than rejecting it with FORG0006.
func TestSchemaDerivedDurationAggregate(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	parsed, err := xpath3.CastFromString("PT11S", xpath3.TypeDayTimeDuration)
	require.NoError(t, err)
	derived := xpath3.AtomicValue{
		TypeName: "Q{urn:test}dtd",
		BaseType: xpath3.TypeDayTimeDuration,
		Value:    parsed.Value,
	}

	vars := xpath3.NewVariables()
	vars.Set("d", xpath3.SingleAtomic(derived))

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(vars)

	sumSeq := evalExprWithEval(t, eval, doc, `sum(($d, $d))`)
	require.Equal(t, 1, sumSeq.Len())
	sumStr, err := xpath3.AtomicToString(sumSeq.Get(0).(xpath3.AtomicValue))
	require.NoError(t, err)
	require.Equal(t, "PT22S", sumStr)

	avgSeq := evalExprWithEval(t, eval, doc, `avg(($d, $d))`)
	require.Equal(t, 1, avgSeq.Len())
	avgStr, err := xpath3.AtomicToString(avgSeq.Get(0).(xpath3.AtomicValue))
	require.NoError(t, err)
	require.Equal(t, "PT11S", avgStr)
}

package xpath3_test

import (
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

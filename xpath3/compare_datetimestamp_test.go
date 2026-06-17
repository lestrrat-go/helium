package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestEvalDateTimeStampComparison covers comparisons involving xs:dateTimeStamp.
// xs:dateTimeStamp is a recognized XSD 1.1 type and is a subtype of xs:dateTime,
// so it must compare like xs:dateTime both against itself and against xs:dateTime.
func TestEvalDateTimeStampComparison(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	tests := []struct {
		expr   string
		expect bool
	}{
		// same-type dateTimeStamp comparisons
		{`xs:dateTimeStamp("2020-01-01T00:00:00Z") eq xs:dateTimeStamp("2020-01-01T00:00:00Z")`, true},
		{`xs:dateTimeStamp("2020-01-01T00:00:00Z") ne xs:dateTimeStamp("2021-01-01T00:00:00Z")`, true},
		{`xs:dateTimeStamp("2020-01-01T00:00:00Z") lt xs:dateTimeStamp("2021-01-01T00:00:00Z")`, true},
		{`xs:dateTimeStamp("2021-01-01T00:00:00Z") gt xs:dateTimeStamp("2020-01-01T00:00:00Z")`, true},
		{`xs:dateTimeStamp("2020-01-01T00:00:00Z") le xs:dateTimeStamp("2020-01-01T00:00:00Z")`, true},
		{`xs:dateTimeStamp("2020-01-01T00:00:00Z") ge xs:dateTimeStamp("2020-01-01T00:00:00Z")`, true},
		// mixed dateTime / dateTimeStamp comparisons
		{`xs:dateTime("2020-01-01T00:00:00Z") eq xs:dateTimeStamp("2020-01-01T00:00:00Z")`, true},
		{`xs:dateTimeStamp("2020-01-01T00:00:00Z") eq xs:dateTime("2020-01-01T00:00:00Z")`, true},
		{`xs:dateTimeStamp("2020-01-01T00:00:00Z") lt xs:dateTime("2021-01-01T00:00:00Z")`, true},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
			require.NoError(t, err)
			seq := result.Sequence()
			require.Equal(t, 1, seq.Len())
			av := seq.Get(0).(xpath3.AtomicValue)
			require.Equal(t, tc.expect, av.BooleanVal())
		})
	}
}

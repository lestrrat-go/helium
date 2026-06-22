package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestFormatNumber_Variants(t *testing.T) {
	// Empty-sequence first argument -> NaN per F&O.
	r, err := evaluate(t.Context(), nil, `format-number((), "0")`)
	require.NoError(t, err)
	require.Equal(t, "NaN", r.StringValue())

	// untypedAtomic first arg is cast to double.
	r, err = evaluate(t.Context(), nil, `format-number(xs:untypedAtomic("12.5"), "0.0")`)
	require.NoError(t, err)
	require.Equal(t, "12.5", r.StringValue())

	// Basic picture strings.
	for _, tc := range []struct{ expr, want string }{
		{`format-number(1234.5, "#,##0.00")`, "1,234.50"},
		{`format-number(0.25, "0%")`, "25%"},
		{`format-number(-3, "0;(0)")`, "(3)"},
		{`format-number(1234, "0.0e0")`, "1.2e3"},
	} {
		r, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		require.Equal(t, tc.want, r.StringValue(), tc.expr)
	}

	// Non-numeric first arg -> XPTY0004.
	_, err = evaluate(t.Context(), nil, `format-number(current-date(), "0")`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
}

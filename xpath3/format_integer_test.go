package xpath3_test

import (
	"testing"
	"time"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// fn:format-integer($value as xs:integer?, $picture as xs:string, ...).
// $picture is a required, exactly-one xs:string. It must be validated
// regardless of whether $value is empty — an empty or oversized $picture
// is XPTY0004, never a panic, and an empty $value with a valid $picture
// yields a zero-length string.
func TestFormatIntegerPictureValidation(t *testing.T) {
	t.Parallel()

	t.Run("empty picture with non-empty value raises XPTY0004", func(t *testing.T) {
		_, err := evaluate(t.Context(), nil, `format-integer(1, ())`)
		require.Error(t, err)
		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr)
		require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
	})

	t.Run("empty picture with empty value raises XPTY0004", func(t *testing.T) {
		_, err := evaluate(t.Context(), nil, `format-integer((), ())`)
		require.Error(t, err)
		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr)
		require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
	})

	t.Run("oversized picture raises XPTY0004", func(t *testing.T) {
		_, err := evaluate(t.Context(), nil, `format-integer(1, ("0", "0"))`)
		require.Error(t, err)
		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr)
		require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
	})

	t.Run("empty value with valid picture yields empty string", func(t *testing.T) {
		result, err := evaluate(t.Context(), nil, `format-integer((), "0")`)
		require.NoError(t, err)
		require.Equal(t, "", result.StringValue())
	})

	t.Run("normal value still formats", func(t *testing.T) {
		result, err := evaluate(t.Context(), nil, `format-integer(123, "0")`)
		require.NoError(t, err)
		require.Equal(t, "123", result.StringValue())
	})
}

// XPath function conversion atomizes an argument before checking cardinality.
// A picture such as ([], "0") atomizes to the single string "0", so it must be
// accepted, not rejected as a 2-item sequence. This is handled by the shared
// xs:string coercion helper, so format-number must behave identically.
func TestStringArgAtomizesBeforeCardinality(t *testing.T) {
	t.Parallel()

	cases := []struct {
		expr   string
		expect string
	}{
		{`format-integer(123, ([], "0"))`, "123"},
		{`format-integer((), ([], "0"))`, ""},
		{`format-number(123, ([], "0"))`, "123"},
		{`format-number((), ([], "0.0"))`, "NaN"},
		{`upper-case(([], "ab"))`, "AB"},
		{`concat(([], "x"), "y")`, "xy"},
	}
	for _, tc := range cases {
		result, err := evaluate(t.Context(), nil, tc.expr)
		require.NoError(t, err, tc.expr)
		require.Equal(t, tc.expect, result.StringValue(), tc.expr)
	}

	// A genuinely oversized atomized picture is still XPTY0004.
	_, err := evaluate(t.Context(), nil, `format-integer(1, ("0", "0"))`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
}

// The atomize-first rewrite must not materialize a large sequence just to reject
// it on cardinality — it stops at the second atomized item.
func TestStringArgRejectsLongSequencePromptly(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := evaluate(t.Context(), nil, `upper-case(1 to 100000000)`)
		require.Error(t, err)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("coercion materialized a large sequence instead of stopping at the 2nd item")
	}
}

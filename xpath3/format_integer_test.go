package xpath3_test

import (
	"testing"

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

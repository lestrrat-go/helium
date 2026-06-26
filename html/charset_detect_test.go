package html_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// The charset prescan must recognize the WHATWG meta-charset value forms beyond
// the bare double-quoted/unquoted ones: single-quoted values and values with
// ASCII whitespace around "=". When such a form declares utf-8, an invalid high
// byte must be replaced with U+FFFD and raise the encoding-error diagnostic,
// rather than silently falling back to the Latin-1/Windows-1252 decode.
func TestCharsetDetect_WHATWGMetaForms(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		meta string
	}{
		{name: "single-quoted", meta: `<meta charset='utf-8'>`},
		{name: "space-around-equals", meta: `<meta charset = utf-8>`},
		{name: "single-quoted-space", meta: `<meta charset = 'utf-8'>`},
		{name: "double-quoted-baseline", meta: `<meta charset="utf-8">`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 0xFF is invalid UTF-8. With utf-8 detected it becomes U+FFFD and
			// raises an encoding error; without detection it would be decoded as
			// Latin-1 'ÿ' (U+00FF) with no error.
			input := []byte("<html><head>" + tc.meta + "</head><body>x\xFFy</body></html>")

			var chars strings.Builder
			var sawEncodingError bool
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(html.CharactersFunc(func(data []byte) error {
				chars.Write(data)
				return nil
			}))
			sax.SetOnError(html.ErrorFunc(func(err error) error {
				if err != nil && err.Error() == "Invalid bytes in character encoding" {
					sawEncodingError = true
				}
				return nil
			}))

			err := html.NewParser().ParseWithSAX(t.Context(), input, sax)
			require.NoError(t, err)
			require.True(t, sawEncodingError,
				"declared utf-8 must raise the encoding-error diagnostic for the invalid byte")
			require.Contains(t, chars.String(), "�",
				"invalid byte under declared utf-8 must become U+FFFD")
			require.NotContains(t, chars.String(), "ÿ",
				"invalid byte must not fall through to the Latin-1 decode")
		})
	}
}

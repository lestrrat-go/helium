package html

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// extractDeclaredCharset must yield a clean encoding name even when the charset
// value sits inside a still-quoted content attribute. The byte after "charset="
// is then an ordinary letter (not a quote), so the unquoted-value scan must
// terminate at the enclosing quote rather than swallowing it. (Regression:
// PR #812 / HTML-004 — previously returned `utf-8"`/`iso-8859-1"`.)
func TestExtractDeclaredCharset(t *testing.T) {
	t.Parallel()

	const utf8 = "utf-8"
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{name: "bare-unquoted", in: `<meta charset=utf-8>`, want: utf8},
		{name: "double-quoted", in: `<meta charset="utf-8">`, want: utf8},
		{name: "single-quoted", in: `<meta charset='utf-8'>`, want: utf8},
		{name: "space-around-equals", in: `<meta charset = utf-8>`, want: utf8},
		{
			name: "http-equiv-content-type-utf8",
			in:   `<meta http-equiv="Content-Type" content="text/html; charset=utf-8">`,
			want: utf8,
		},
		{
			name: "http-equiv-content-type-iso",
			in:   `<meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1">`,
			want: "iso-8859-1",
		},
		{
			name: "http-equiv-content-type-single-quoted",
			in:   `<meta http-equiv='Content-Type' content='text/html; charset=iso-8859-1'>`,
			want: "iso-8859-1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, extractDeclaredCharset([]byte(tc.in)))
		})
	}
}

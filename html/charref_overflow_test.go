package html_test

import (
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestNumericCharRefOverflow verifies that a numeric character reference whose
// value overflows a 32-bit integer or exceeds the maximum Unicode code point is
// mapped to the replacement character U+FFFD per HTML5, rather than being
// dropped entirely (the buggy behavior treated ParseInt overflow as "no
// digits" and emitted nothing). Both element-text and attribute-value contexts
// must produce U+FFFD.
func TestNumericCharRefOverflow(t *testing.T) {
	const repl = "�"

	cases := map[string]struct {
		input string
		want  string // expected serialized substring containing the U+FFFD site
	}{
		// Decimal value far above the 32-bit signed range.
		"text-decimal-overflow": {"<p>x&#9999999999;y</p>", "<p>x" + repl + "y</p>"},
		// Hex value just above the max code point (0x10FFFF).
		"text-hex-above-max": {"<p>x&#x110000;y</p>", "<p>x" + repl + "y</p>"},
		// Enormous hex value that overflows the accumulator.
		"text-hex-overflow": {"<p>x&#xFFFFFFFFFF;y</p>", "<p>x" + repl + "y</p>"},
		// Same in attribute values.
		"attr-decimal-overflow": {`<p title="x&#9999999999;y">z</p>`, `title="x` + repl + `y"`},
		"attr-hex-above-max":    {`<p title="x&#x110000;y">z</p>`, `title="x` + repl + `y"`},
		"attr-hex-overflow":     {`<p title="x&#xFFFFFFFFFF;y">z</p>`, `title="x` + repl + `y"`},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			doc, err := html.NewParser().
				SuppressErrors(true).
				SuppressWarnings(true).
				Parse(t.Context(), []byte(tc.input))
			require.NoError(t, err, "parse must not error")

			out, err := html.WriteString(doc)
			require.NoError(t, err, "serialize must not error")
			require.Contains(t, out, tc.want, "out-of-range numeric char ref must yield U+FFFD: %q", out)
		})
	}
}

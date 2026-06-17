package html_test

import (
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestNULCharacterNoHang guards against an infinite loop when a real U+0000
// (NUL) byte appears in character data. The cursor returns 0 both for EOF and
// for an actual NUL byte, so the scan loops in parseCharacters,
// parseRCDATAContent and parsePlaintext broke with no progress and the parse
// loop spun forever. A watchdog converts a regression into a failure instead of
// stalling the suite.
func TestNULCharacterNoHang(t *testing.T) {
	inputs := map[string][]byte{
		"lone-nul":      {0x00},
		"body-nul":      []byte("<html><body>\x00</body></html>"),
		"text-nul":      []byte("<html><body>a\x00b</body></html>"),
		"nested-nul":    []byte("<p>x\x00\x00y</p>"),
		"title-nul":     []byte("<title>a\x00b</title>"),
		"textarea-nul":  []byte("<textarea>a\x00b</textarea>"),
		"script-nul":    []byte("<script>a\x00b</script>"),
		"style-nul":     []byte("<style>a\x00b</style>"),
		"plaintext-nul": []byte("<plaintext>a\x00b"),
		// NUL immediately after an end-tag name must not hang either.
		"end-tag-nul": []byte("</title\x00"),
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			done := make(chan struct{})
			go func() {
				defer close(done)
				// We don't care about the result, only that it returns.
				_, _ = html.NewParser().Parse(ctx, input)
			}()

			watchdog := time.NewTimer(3 * time.Second)
			defer watchdog.Stop()

			select {
			case <-done:
			case <-watchdog.C:
				t.Fatal("parse hang detected (NUL byte infinite loop)")
			}
		})
	}
}

// TestNULCharacterReplaced verifies that a real U+0000 byte is replaced with
// the U+FFFD replacement character (per HTML5 data/RCDATA/RAWTEXT/PLAINTEXT
// state handling) across every content mode, rather than being dropped, kept
// literally, or truncating the surrounding content.
func TestNULCharacterReplaced(t *testing.T) {
	const repl = "�"

	cases := map[string]struct {
		input string
		want  string // expected serialized substring containing the NUL site
	}{
		"data":      {"<html><body>a\x00b</body></html>", "<body>a" + repl + "b</body>"},
		"rcdata":    {"<title>a\x00b</title>", "<title>a" + repl + "b</title>"},
		"textarea":  {"<textarea>a\x00b</textarea>", "<textarea>a" + repl + "b</textarea>"},
		"rawscript": {"<script>a\x00b</script>", "<script>a" + repl + "b</script>"},
		"rawstyle":  {"<style>a\x00b</style>", "<style>a" + repl + "b</style>"},
		"plaintext": {"<plaintext>a\x00b", "<plaintext>a" + repl + "b</plaintext>"},
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
			require.NotContains(t, out, "\x00", "literal NUL must not survive")
			require.Contains(t, out, tc.want, "NUL must be replaced with U+FFFD")
			// The bytes surrounding the NUL must be preserved (no truncation).
			require.True(t, strings.Contains(out, "a"+repl+"b"), "content around NUL must be intact: %q", out)
		})
	}
}

package lexer_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/xpath1/lexer"
	"github.com/stretchr/testify/require"
)

func TestDiagnosticExcerpt(t *testing.T) {
	t.Run("short text is unchanged", func(t *testing.T) {
		const input = "short 世界"
		require.Equal(t, input, lexer.DiagnosticExcerpt(input))
	})

	t.Run("ASCII is bounded", func(t *testing.T) {
		within := strings.Repeat("a", lexer.DiagnosticExcerptByteLimit)
		require.Equal(t, within, lexer.DiagnosticExcerpt(within))

		got := lexer.DiagnosticExcerpt(within + "a")
		require.LessOrEqual(t, len(got), lexer.DiagnosticExcerptByteLimit)
		require.True(t, utf8.ValidString(got))
		require.Contains(t, got, "[truncated]")
	})

	t.Run("multibyte text ends on a rune boundary", func(t *testing.T) {
		within := strings.Repeat("界", lexer.DiagnosticExcerptByteLimit/len("界"))
		require.Equal(t, within, lexer.DiagnosticExcerpt(within))

		got := lexer.DiagnosticExcerpt(within + "界")
		require.LessOrEqual(t, len(got), lexer.DiagnosticExcerptByteLimit)
		require.True(t, utf8.ValidString(got))
		require.Contains(t, got, "[truncated]")
	})

	t.Run("invalid UTF-8 is replaced", func(t *testing.T) {
		got := lexer.DiagnosticExcerpt(string([]byte{'a', 0xff, 'b'}))
		require.Equal(t, "a\uFFFDb", got)
		require.True(t, utf8.ValidString(got))
	})
}

func TestTokenStringDiagnosticExcerpt(t *testing.T) {
	t.Run("short quoted value is unchanged", func(t *testing.T) {
		tok := lexer.Token{Type: lexer.TokenString, Value: "a\n\"b"}
		require.Equal(t, `String("a\n\"b")`, tok.String())
	})

	t.Run("escaped value is bounded", func(t *testing.T) {
		tok := lexer.Token{Type: lexer.TokenString, Value: strings.Repeat("\x00", 1<<20)}
		got := tok.String()
		require.LessOrEqual(t, len(got), 4*lexer.DiagnosticExcerptByteLimit+32)
		require.True(t, utf8.ValidString(got))
		require.Contains(t, got, "[truncated]")
	})
}

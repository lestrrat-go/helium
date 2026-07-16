package helium_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestParserZeroValue verifies that a zero-value helium.Parser behaves
// identically to one returned by helium.NewParser() — same secure defaults,
// same parse results, and usable both directly and as the head of an
// option-method chain, without panicking on its nil config.
func TestParserZeroValue(t *testing.T) {
	serialize := func(t *testing.T, doc *helium.Document) string {
		t.Helper()
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	t.Run("Parse matches NewParser", func(t *testing.T) {
		const src = `<root att="v"><child>text</child><child/></root>`
		var zero helium.Parser
		zdoc, err := zero.Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		ndoc, err := helium.NewParser().Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		require.Equal(t, serialize(t, ndoc), serialize(t, zdoc))
	})

	t.Run("ParseReader does not panic", func(t *testing.T) {
		var zero helium.Parser
		doc, err := zero.ParseReader(context.Background(), strings.NewReader(`<r><a/></r>`))
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("option chaining works and matches NewParser", func(t *testing.T) {
		const src = `<r>  <a/>  </r>`
		var zero helium.Parser
		zdoc, err := zero.StripBlanks(true).Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		ndoc, err := helium.NewParser().StripBlanks(true).Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		require.Equal(t, serialize(t, ndoc), serialize(t, zdoc))
	})

	t.Run("secure defaults: element depth cap matches NewParser", func(t *testing.T) {
		// NewParser caps nesting at 256; a zero-value Parser must apply the same
		// cap, so a document nested well past it fails for both, identically.
		var b strings.Builder
		const depth = 400
		for range depth {
			b.WriteString("<a>")
		}
		for range depth {
			b.WriteString("</a>")
		}
		src := []byte(b.String())

		var zero helium.Parser
		_, zerr := zero.Parse(context.Background(), src)
		_, nerr := helium.NewParser().Parse(context.Background(), src)
		require.Error(t, nerr, "NewParser should reject nesting past its depth cap")
		require.Error(t, zerr, "zero-value Parser must apply the same depth cap")
	})
}

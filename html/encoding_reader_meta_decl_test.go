package html_test

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"

	"github.com/stretchr/testify/require"
)

// TestNonDeclarationCharsetStaysUTF8 guards the regression where a valid UTF-8
// document that merely CONTAINS the text "charset=iso-8859-1" outside of a real
// <meta> element (in ordinary text, a comment, or a script) was eagerly decoded
// as Latin-1 and corrupted (café -> cafÃ©). The eager Latin-1/UTF-8 commit
// overrides utf8.Valid, so it must trust ONLY an actual <meta> charset
// declaration, not any charset= token in the first 1024 bytes. Both Parse and
// ParseReader must leave such a document as UTF-8.
func TestNonDeclarationCharsetStaysUTF8(t *testing.T) {
	t.Parallel()

	serialize := func(d *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, html.NewWriter().WriteTo(&buf, d))
		return buf.String()
	}
	textOf := func(d *helium.Document) string {
		var text bytes.Buffer
		for n := range helium.Descendants(d) {
			if tx, ok := n.(*helium.Text); ok {
				text.Write(tx.Content())
			}
		}
		return text.String()
	}

	for _, tc := range []struct {
		name string
		doc  []byte
	}{
		{
			name: "text",
			doc:  []byte("<html><body><p>charset=iso-8859-1 caf\xC3\xA9</p></body></html>"),
		},
		{
			name: "comment",
			doc:  []byte("<html><body><!-- see charset=iso-8859-1 --><p>caf\xC3\xA9</p></body></html>"),
		},
		{
			name: "script-element",
			doc:  []byte(`<html><head><script>var c="charset=iso-8859-1";</script></head><body><p>caf` + "\xC3\xA9" + `</p></body></html>`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.True(t, utf8.Valid(tc.doc), "test input must be valid UTF-8")

			bytesDoc, err := html.NewParser().Parse(t.Context(), tc.doc)
			require.NoError(t, err)
			require.NotContains(t, textOf(bytesDoc), "Ã©",
				"Parse must not corrupt valid UTF-8 over a non-declaration charset= token")
			require.NotContains(t, bytesDoc.Encoding(), "8859",
				"Parse must not commit to ISO-8859-1 without a real meta declaration")

			readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(tc.doc))
			require.NoError(t, err)
			require.Equal(t, serialize(bytesDoc), serialize(readerDoc),
				"Parse and ParseReader must agree (both stay UTF-8)")
			require.NotContains(t, readerDoc.Encoding(), "8859",
				"ParseReader must not commit to ISO-8859-1 without a real meta declaration")
		})
	}
}

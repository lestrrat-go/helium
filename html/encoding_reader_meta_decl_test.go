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

// TestMetaCharsetAttributePrecisionParity guards the WHATWG per-meta attribute
// rule end-to-end through Parse and ParseReader. A `charset=iso-8859-1` token in
// the WRONG attribute (data-charset, or a content value with no
// http-equiv="content-type") must NOT commit the document to Latin-1: a valid
// UTF-8 body stays UTF-8 (café, not cafÃ©). A real charset attribute or a
// content-type pragma DOES commit to Latin-1. Both APIs must agree.
func TestMetaCharsetAttributePrecisionParity(t *testing.T) {
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

	// cafe with a UTF-8 e-acute (\xC3\xA9). Latin-1 misdecoding yields "Ã©".
	const cafeUTF8 = "caf\xC3\xA9"

	for _, tc := range []struct {
		name      string
		meta      string
		wantLatin bool // true: should decode as Latin-1 (corrupt the UTF-8 body)
	}{
		{
			name: "data-charset-ignored",
			meta: `<meta data-charset=iso-8859-1>`,
		},
		{
			name: "non-pragma-content-ignored",
			meta: `<meta name=description content="charset=iso-8859-1">`,
		},
		{
			name:      "real-charset-attr-latin1",
			meta:      `<meta charset=iso-8859-1>`,
			wantLatin: true,
		},
		{
			name:      "pragma-content-type-latin1",
			meta:      `<meta http-equiv="content-type" content="text/html; charset=iso-8859-1">`,
			wantLatin: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := []byte("<html><head>" + tc.meta + "</head><body><p>" + cafeUTF8 + "</p></body></html>")
			require.True(t, utf8.Valid(doc), "test input must be valid UTF-8")

			bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
			require.NoError(t, err)
			readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
			require.NoError(t, err)

			require.Equal(t, serialize(bytesDoc), serialize(readerDoc),
				"Parse and ParseReader must agree on the encoding decision")

			switch tc.wantLatin {
			case true:
				require.Contains(t, textOf(bytesDoc), "Ã©",
					"a real meta charset declaration must commit to Latin-1")
				require.Contains(t, bytesDoc.Encoding(), "8859",
					"a real meta charset declaration must record ISO-8859-1")
			case false:
				require.NotContains(t, textOf(bytesDoc), "Ã©",
					"charset= in the wrong attribute must not corrupt valid UTF-8")
				require.NotContains(t, bytesDoc.Encoding(), "8859",
					"charset= in the wrong attribute must not commit to ISO-8859-1")
			}
		})
	}
}

package html_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"

	"github.com/stretchr/testify/require"
)

// TestParseReaderDeferredLatin1MatchesParse guards encoding parity for an
// undeclared-charset HTML document that is valid UTF-8 up front (it contains a
// genuine UTF-8 multibyte sequence) but then carries a raw Latin-1 byte later.
//
// The whole-document Parse([]byte) path decides the encoding for the ENTIRE
// document at once: because the document as a whole is not valid UTF-8, every
// byte — including the leading valid UTF-8 multibyte sequence — is reinterpreted
// as Windows-1252. The streaming ParseReader/ParseFile path must reach the same
// result rather than passing the early UTF-8 bytes through verbatim and only
// converting from the first invalid byte onward.
func TestParseReaderDeferredLatin1MatchesParse(t *testing.T) {
	t.Parallel()

	// "café" with é as a real UTF-8 sequence (0xC3 0xA9), then a lone raw
	// Latin-1 0xE9 ('é') which makes the document as a whole invalid UTF-8.
	doc := []byte("<html><body><p>caf\xC3\xA9 \xE9</p></body></html>")
	require.False(t, utf8.Valid(doc), "test input must not be valid UTF-8 as a whole")

	serialize := func(d *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, html.NewWriter().WriteTo(&buf, d))
		return buf.String()
	}

	bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
	require.NoError(t, err)
	want := serialize(bytesDoc)
	wantEnc := bytesDoc.Encoding()
	require.Equal(t, "Windows-1252", wantEnc,
		"the []byte path must reinterpret the whole document as Windows-1252")

	readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
	require.NoError(t, err)
	require.Equal(t, want, serialize(readerDoc),
		"ParseReader output must match Parse([]byte)")
	require.Equal(t, wantEnc, readerDoc.Encoding(),
		"ParseReader must detect the same encoding as Parse([]byte)")

	dir := t.TempDir()
	path := filepath.Join(dir, "doc.html")
	require.NoError(t, os.WriteFile(path, doc, 0o600))
	fileDoc, err := html.NewParser().ParseFile(t.Context(), path)
	require.NoError(t, err)
	require.Equal(t, want, serialize(fileDoc),
		"ParseFile output must match Parse([]byte)")
	require.Equal(t, wantEnc, fileDoc.Encoding(),
		"ParseFile must detect the same encoding as Parse([]byte)")
}

// TestParseReaderRuneStraddlesSniffBoundary guards against misclassifying a
// fully-valid UTF-8 document as Latin-1/Windows-1252 when a multibyte rune
// straddles the 1024-byte charset sniff boundary.
//
// The sniff window reads exactly 1024 bytes. If byte 1024 splits a valid
// multibyte rune (e.g. é = 0xC3 0xA9), a naive utf8.Valid(head) check reports
// the prefix as invalid and reinterprets the whole document as Windows-1252,
// corrupting every multibyte rune (é → Ã©). A genuine valid-UTF-8 document must
// instead stay UTF-8 and parse identically via Parse([]byte), ParseReader, and
// ParseFile.
func TestParseReaderRuneStraddlesSniffBoundary(t *testing.T) {
	t.Parallel()

	// Build a valid-UTF-8 document where an 'é' (0xC3 0xA9) is positioned so the
	// 0xC3 lead byte lands at offset 1023 and the 0xA9 continuation at 1024,
	// straddling the sniff boundary.
	var b bytes.Buffer
	b.WriteString("<html><body><p>")
	for b.Len() < 1023 {
		b.WriteByte('a')
	}
	require.Equal(t, 1023, b.Len(), "filler must place the rune lead byte at offset 1023")
	b.WriteString("é") // 0xC3 0xA9 straddles offsets 1023/1024
	b.WriteString("more café text</p></body></html>")
	doc := b.Bytes()
	require.True(t, utf8.Valid(doc), "test input must be valid UTF-8 as a whole")

	serialize := func(d *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, html.NewWriter().WriteTo(&buf, d))
		return buf.String()
	}

	bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
	require.NoError(t, err)
	want := serialize(bytesDoc)
	require.NotContains(t, want, "Ã©", "Parse([]byte) must not corrupt the straddling rune")

	readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
	require.NoError(t, err)
	require.Equal(t, want, serialize(readerDoc),
		"ParseReader output must match Parse([]byte) for a boundary-straddling rune")

	dir := t.TempDir()
	path := filepath.Join(dir, "doc.html")
	require.NoError(t, os.WriteFile(path, doc, 0o600))
	fileDoc, err := html.NewParser().ParseFile(t.Context(), path)
	require.NoError(t, err)
	require.Equal(t, want, serialize(fileDoc),
		"ParseFile output must match Parse([]byte) for a boundary-straddling rune")

	// None of the paths should switch to a Latin-1/Windows-1252 encoding.
	require.NotContains(t, readerDoc.Encoding(), "1252")
	require.NotContains(t, readerDoc.Encoding(), "8859")
	require.NotContains(t, fileDoc.Encoding(), "1252")
	require.NotContains(t, fileDoc.Encoding(), "8859")
}

// TestParseReaderAllValidUTF8StaysUTF8 confirms the deferred path still leaves
// a fully-valid-UTF8 undeclared document as UTF-8 (no spurious Latin-1 switch).
func TestParseReaderAllValidUTF8StaysUTF8(t *testing.T) {
	t.Parallel()

	doc := []byte("<html><body><p>caf\xC3\xA9 na\xC3\xAFve</p></body></html>")
	require.True(t, utf8.Valid(doc))

	serialize := func(d *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, html.NewWriter().WriteTo(&buf, d))
		return buf.String()
	}

	bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
	require.NoError(t, err)
	want := serialize(bytesDoc)

	readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
	require.NoError(t, err)
	require.Equal(t, want, serialize(readerDoc),
		"valid-UTF8 document must serialize identically via both paths")
	// The deferred reader must not switch to a Latin-1/Windows-1252 encoding for
	// an all-valid-UTF8 document. (The exact UTF-8 encoding-name reporting
	// between the []byte and reader paths is a separate, pre-existing concern;
	// here we only assert no spurious Latin-1 switch.)
	require.NotContains(t, readerDoc.Encoding(), "1252")
	require.NotContains(t, readerDoc.Encoding(), "8859")
}

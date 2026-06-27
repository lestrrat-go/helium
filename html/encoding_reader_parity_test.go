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

// TestParseReaderDeclaredLatin1PastCapStaysISO88591 guards that an explicit
// charset=iso-8859-1 declaration streams straight through the Latin-1 reader and
// is NOT routed through the deferred/bounded path. The first high byte (0xE9 =
// 'é' in ISO-8859-1) lands well past the deferred reader's 1 MiB commit cap: if
// the declared stream went through that path, the cap would commit it to UTF-8
// and sanitize the 0xE9 to U+FFFD, discarding the declared encoding. A declared
// encoding must decode faithfully regardless of how far in its first high byte
// appears.
func TestParseReaderDeclaredLatin1PastCapStaysISO88591(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	b.WriteString("<html><head><meta charset=iso-8859-1></head><body><p>")
	for b.Len() < (1<<20)+4096 { // past the deferred reader's 1 MiB commit cap
		b.WriteByte('a')
	}
	b.WriteByte(0xE9) // 'é' in ISO-8859-1
	b.WriteString("</p></body></html>")

	doc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(b.Bytes()))
	require.NoError(t, err)
	require.Equal(t, "ISO-8859-1", doc.Encoding(),
		"a declared charset=iso-8859-1 stream must report ISO-8859-1, not commit to UTF-8")

	var text bytes.Buffer
	for n := range helium.Descendants(doc) {
		if t, ok := n.(*helium.Text); ok {
			text.Write(t.Content())
		}
	}
	require.Contains(t, text.String(), "é",
		"the late 0xE9 must decode as ISO-8859-1 'é' (UTF-8 in the DOM)")
	require.NotContains(t, text.String(), "�",
		"a declared Latin-1 byte must never be sanitized to U+FFFD")
}

// TestParseReaderDeferredCommitInvalidByteRaisesEncodingError covers the
// post-commit sanitizer path: an UNDECLARED stream that stays valid UTF-8 past
// the deferred reader's 1 MiB cap (so the reader commits to UTF-8) and THEN
// carries a raw non-UTF-8 byte. The byte is sanitized to U+FFFD, and the parser
// must raise the same "Invalid bytes in character encoding" SAX diagnostic the
// declared-UTF-8 sanitizer path emits — not silently swallow it.
func TestParseReaderDeferredCommitInvalidByteRaisesEncodingError(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	b.WriteString("<html><body><p>")
	for b.Len() < (1<<20)+4096 { // commit to UTF-8 at the 1 MiB cap
		b.WriteByte('a')
	}
	b.WriteByte(0x93) // lone Windows-1252 byte: invalid UTF-8, post-commit
	b.WriteString("z</p></body></html>")

	var chars bytes.Buffer
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

	pp := html.NewParser().NewSAXPushParser(t.Context(), sax)
	require.NoError(t, pp.Push(b.Bytes()))
	_, err := pp.Close()
	require.NoError(t, err)

	require.True(t, sawEncodingError,
		"a post-commit invalid byte must raise the encoding-error diagnostic")
	require.Contains(t, chars.String(), "�",
		"the post-commit invalid byte must be sanitized to U+FFFD")
	require.NotContains(t, chars.String(), "\x93",
		"the raw invalid byte must never leak into SAX char data")
}

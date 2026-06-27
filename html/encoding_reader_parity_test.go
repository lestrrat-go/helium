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

// TestDeclaredLatin1ParseVsParseReaderParity guards the API-parity fix for a
// DECLARED charset=iso-8859-1 document whose bytes are ALSO valid UTF-8. The
// streaming ParseReader path commits to Latin-1 on the declaration immediately,
// so the in-memory Parse([]byte) path must do the same — decoding the bytes
// 0xC3 0xA9 as two Latin-1 chars ("Ã©"), NOT as the single UTF-8 rune "é" it
// happens to form. Before the fix Parse honored the declaration only when the
// bytes were INvalid UTF-8, so the two APIs diverged on this input.
func TestDeclaredLatin1ParseVsParseReaderParity(t *testing.T) {
	t.Parallel()

	// The whole document is valid UTF-8 ("caf" + the UTF-8 sequence for é), yet it
	// declares charset=iso-8859-1, so both APIs must interpret it as Latin-1.
	doc := []byte("<html><head><meta charset=iso-8859-1></head><body><p>caf\xC3\xA9</p></body></html>")
	require.True(t, utf8.Valid(doc), "test input must be valid UTF-8 as a whole")

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

	bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, "ISO-8859-1", bytesDoc.Encoding(),
		"declared charset=iso-8859-1 must be honored even when the bytes are valid UTF-8")
	require.Contains(t, textOf(bytesDoc), "Ã©",
		"the bytes 0xC3 0xA9 must decode as two Latin-1 chars, not one UTF-8 rune")

	readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
	require.NoError(t, err)
	require.Equal(t, serialize(bytesDoc), serialize(readerDoc),
		"Parse([]byte) and ParseReader must agree for declared Latin-1 input")
	require.Equal(t, bytesDoc.Encoding(), readerDoc.Encoding(),
		"both APIs must report the same declared ISO-8859-1 encoding")
}

// TestDeclaredLatin1QuotedGTInMetaParity guards the meta-prescan against a '>'
// that sits inside a QUOTED attribute value before charset=. A naive scan that
// bounds the meta tag at the first '>' byte would truncate
// <meta data-x=">" charset="iso-8859-1"> before charset=, miss the declaration,
// and decode the valid-UTF-8 bytes as UTF-8 instead of declared Latin-1. The
// prescan must find the first UNQUOTED '>', so both Parse and ParseReader honor
// the charset=iso-8859-1 declaration.
func TestDeclaredLatin1QuotedGTInMetaParity(t *testing.T) {
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
		meta string
	}{
		{
			name: "charset-after-quoted-gt",
			meta: `<meta data-x=">" charset="iso-8859-1">`,
		},
		{
			name: "http-equiv-quoted-gt",
			meta: `<meta data-x=">" http-equiv="Content-Type" content="text/html; charset=iso-8859-1">`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// The whole document is valid UTF-8 ("caf" + the UTF-8 é sequence),
			// yet it declares iso-8859-1, so both APIs must decode as Latin-1.
			doc := []byte("<html><head>" + tc.meta + "</head><body><p>caf\xC3\xA9</p></body></html>")
			require.True(t, utf8.Valid(doc), "test input must be valid UTF-8 as a whole")

			bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
			require.NoError(t, err)
			require.Equal(t, "ISO-8859-1", bytesDoc.Encoding(),
				"a quoted '>' before charset= must not hide the iso-8859-1 declaration")
			require.Contains(t, textOf(bytesDoc), "Ã©",
				"the bytes 0xC3 0xA9 must decode as two Latin-1 chars, not one UTF-8 rune")

			readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
			require.NoError(t, err)
			require.Equal(t, serialize(bytesDoc), serialize(readerDoc),
				"Parse([]byte) and ParseReader must agree for a quoted-'>' meta tag")
			require.Equal(t, bytesDoc.Encoding(), readerDoc.Encoding(),
				"both APIs must report the same declared ISO-8859-1 encoding")
		})
	}
}

// TestDeclaredLatin1LiteralLessThanBeforeMetaParity guards the meta-prescan
// against a literal non-tag '<' that appears before a real <meta charset=...>
// within the first 1024 bytes. A '<' begins markup only when the byte after it is
// '/', '!', '?', or an ASCII letter; `< " >` or `<x="` is character data. A scan
// that treated such a '<' as a tag and entered quote state on its '"' would ignore
// every later '>' and swallow the genuine <meta charset=iso-8859-1>, decoding the
// valid-UTF-8 bytes as UTF-8 instead of declared Latin-1. The prescan must mirror
// the main parser's char-data rule, so both Parse and ParseReader honor iso-8859-1.
func TestDeclaredLatin1LiteralLessThanBeforeMetaParity(t *testing.T) {
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
		head string
	}{
		{
			name: "quote-bearing-non-tag",
			head: `< " ><meta charset="iso-8859-1">`,
		},
		{
			name: "lt-equals-quote",
			head: `< x="><meta charset=iso-8859-1>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// The whole document is valid UTF-8 ("caf" + the UTF-8 é sequence),
			// yet it declares iso-8859-1, so both APIs must decode as Latin-1.
			doc := []byte("<html><head>" + tc.head + "</head><body><p>caf\xC3\xA9</p></body></html>")
			require.True(t, utf8.Valid(doc), "test input must be valid UTF-8 as a whole")

			bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
			require.NoError(t, err)
			require.Equal(t, "ISO-8859-1", bytesDoc.Encoding(),
				"a literal non-tag '<' before charset= must not hide the iso-8859-1 declaration")
			require.Contains(t, textOf(bytesDoc), "Ã©",
				"the bytes 0xC3 0xA9 must decode as two Latin-1 chars, not one UTF-8 rune")

			readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
			require.NoError(t, err)
			require.Equal(t, serialize(bytesDoc), serialize(readerDoc),
				"Parse([]byte) and ParseReader must agree for a literal-'<'-before-meta doc")
			require.Equal(t, bytesDoc.Encoding(), readerDoc.Encoding(),
				"both APIs must report the same declared ISO-8859-1 encoding")
		})
	}
}

// TestDeclaredCharsetDetectedFromRawBytesParity guards the Parse-vs-ParseReader
// charset-detection window against newline normalization. Parse([]byte) must
// detect the declared charset from the RAW input (first 1024 raw bytes), exactly
// like the streaming ParseReader/push path whose sniff window is read off the raw
// reader BEFORE newline normalization. If Parse instead prescanned the normalized
// bytes, a CRLF-heavy head (each \r\n collapsing to \n) would pull a <meta
// charset=iso-8859-1> that sits PAST raw byte 1024 INTO the post-normalization
// window: Parse would honor the declaration (decode Latin-1, café → cafÃ©) while
// ParseReader never sees it (stays UTF-8) — a parity divergence.
//
// Here ~600 \r\n pairs place the meta past raw byte 1024 but before normalized
// byte 1024, so BOTH APIs must agree, and both must stay UTF-8 (the meta is
// outside the raw 1024-byte window neither path inspects).
func TestDeclaredCharsetDetectedFromRawBytesParity(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	b.WriteString("<html><head>")
	// 600 CRLF pairs = 1200 raw bytes; collapse to 600 LF after normalization.
	for range 600 {
		b.WriteString("\r\n")
	}
	// The meta now starts at raw offset 12+1200 = 1212 (OUTSIDE raw byte 1024)
	// but at normalized offset 12+600 = 612 (INSIDE normalized byte 1024).
	b.WriteString("<meta charset=iso-8859-1>")
	b.WriteString("</head><body><p>caf\xC3\xA9</p></body></html>")
	doc := b.Bytes()
	require.True(t, utf8.Valid(doc), "test input must be valid UTF-8 as a whole")
	require.Greater(t, bytes.Index(doc, []byte("<meta")), 1024,
		"the meta must sit past raw byte 1024 for the divergence to be possible")

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

	bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
	require.NoError(t, err)
	require.NotEqual(t, "ISO-8859-1", bytesDoc.Encoding(),
		"a meta past raw byte 1024 must NOT be detected (matching the streaming prescan)")
	require.Contains(t, textOf(bytesDoc), "é",
		"the bytes 0xC3 0xA9 must stay one UTF-8 rune, not decode as Latin-1 Ã©")

	readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
	require.NoError(t, err)
	require.Equal(t, serialize(bytesDoc), serialize(readerDoc),
		"Parse([]byte) and ParseReader must agree on a meta past raw byte 1024")
	require.Equal(t, bytesDoc.Encoding(), readerDoc.Encoding(),
		"both APIs must report the same encoding for a meta past raw byte 1024")
}

// TestDuplicateCharsetFirstWinsParity guards the WHATWG seen-attribute-name rule
// in the meta prescan: a DUPLICATE attribute name is ignored (first wins). A
// `<meta charset=utf-8 charset=iso-8859-1>` element therefore declares UTF-8, NOT
// Latin-1. Without the rule the later charset would override the earlier one and
// both APIs would commit to Latin-1, corrupting the valid-UTF-8 body (café →
// cafÃ©). Both Parse and ParseReader must keep the document UTF-8 and agree.
func TestDuplicateCharsetFirstWinsParity(t *testing.T) {
	t.Parallel()

	// The whole document is valid UTF-8 ("caf" + the UTF-8 é sequence). The first
	// charset (utf-8) wins, so neither API may reinterpret the bytes as Latin-1.
	doc := []byte("<html><head><meta charset=utf-8 charset=iso-8859-1></head><body><p>caf\xC3\xA9</p></body></html>")
	require.True(t, utf8.Valid(doc), "test input must be valid UTF-8 as a whole")

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

	bytesDoc, err := html.NewParser().Parse(t.Context(), doc)
	require.NoError(t, err)
	require.NotEqual(t, "ISO-8859-1", bytesDoc.Encoding(),
		"a duplicate charset must not override the first; the document stays UTF-8")
	require.Contains(t, textOf(bytesDoc), "é",
		"the bytes 0xC3 0xA9 must stay one UTF-8 rune, not decode as Latin-1 Ã©")

	readerDoc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(doc))
	require.NoError(t, err)
	require.Equal(t, serialize(bytesDoc), serialize(readerDoc),
		"Parse([]byte) and ParseReader must agree for a duplicate-charset meta tag")
	require.Equal(t, bytesDoc.Encoding(), readerDoc.Encoding(),
		"both APIs must report the same encoding for a duplicate-charset meta tag")
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
// 'é' in ISO-8859-1) lands well past the deferred reader's buffering cap (here a
// small MaxContentSize): if the declared stream went through that path it would
// hit the cap still undecided and fail closed with a bounded-input error,
// discarding a perfectly valid declared document. A declared encoding must decode
// faithfully regardless of how far in its first high byte appears.
func TestParseReaderDeclaredLatin1PastCapStaysISO88591(t *testing.T) {
	t.Parallel()

	const limit = 1 << 20 // small content limit so the test stays fast
	var b bytes.Buffer
	b.WriteString("<html><head><meta charset=iso-8859-1></head><body><p>")
	for b.Len() < limit+4096 { // past the deferred reader's commit cap
		b.WriteByte('a')
	}
	b.WriteByte(0xE9) // 'é' in ISO-8859-1
	b.WriteString("</p></body></html>")

	doc, err := html.NewParser().MaxContentSize(limit).ParseReader(t.Context(), bytes.NewReader(b.Bytes()))
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

// TestParseReaderDeferredOverCapFailsClosed covers the bounded-decision
// fail-closed path: an UNDECLARED stream that stays valid UTF-8 past the deferred
// reader's cap (the configured MaxContentSize) and THEN carries a raw non-UTF-8
// byte. The []byte path would reinterpret the whole document as Latin-1, so the
// streaming reader cannot safely commit to UTF-8; rather than silently mis-decode
// the late byte it FAILS CLOSED with ErrContentSizeExceeded. This preserves
// parity (no irreversible mis-decoded SAX/DOM output) and keeps memory bounded.
func TestParseReaderDeferredOverCapFailsClosed(t *testing.T) {
	t.Parallel()

	const limit = 1 << 20 // small content limit so the test stays fast
	var b bytes.Buffer
	b.WriteString("<html><body><p>")
	for b.Len() < limit+4096 { // stay valid UTF-8 past the content limit
		b.WriteByte('a')
	}
	b.WriteByte(0x93) // lone Windows-1252 byte: invalid UTF-8, past the cap
	b.WriteString("z</p></body></html>")

	_, err := html.NewParser().MaxContentSize(limit).ParseReader(t.Context(), bytes.NewReader(b.Bytes()))
	require.ErrorIs(t, err, html.ErrContentSizeExceeded,
		"an undeclared stream that stays valid UTF-8 past the cap must fail closed")
}

// TestParseReaderUndeclaredUTF8UnderLimitParses guards the over-rejection fix:
// an undeclared stream that stays valid UTF-8 well past 1 MiB but below the
// configured content limit (16 MiB default) must PARSE — the deferred reader is
// now bounded by the parser's content limit, not an arbitrary 1 MiB cap, so a
// legitimate ~1.1 MiB ASCII/UTF-8 document is no longer rejected. It settles as
// UTF-8 at EOF and never switches to Latin-1.
func TestParseReaderUndeclaredUTF8UnderLimitParses(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	b.WriteString("<html><body><p>")
	for b.Len() < (1<<20)+(100<<10) { // ~1.1 MiB of valid UTF-8, well under 16 MiB
		b.WriteByte('a')
	}
	b.WriteString("</p></body></html>")
	require.True(t, utf8.Valid(b.Bytes()))

	doc, err := html.NewParser().ParseReader(t.Context(), bytes.NewReader(b.Bytes()))
	require.NoError(t, err,
		"a 1.1 MiB undeclared valid-UTF-8 stream under the content limit must parse")
	require.NotContains(t, doc.Encoding(), "1252",
		"an all-valid-UTF-8 stream must not switch to Windows-1252")
	require.NotContains(t, doc.Encoding(), "8859")
}

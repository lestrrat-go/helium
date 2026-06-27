package html

import (
	"bytes"
	"runtime"
	"strings"
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

// extractMetaCharset must bound the <meta> tag at its first UNQUOTED '>'. A '>'
// inside a quoted attribute value (e.g. <meta data-x=">" charset=iso-8859-1>) is
// part of that value, not the tag terminator, so a naive first-'>' scan would
// truncate the tag before charset= and miss the declaration. (Regression:
// PR #821 / HTML-101.)
func TestExtractMetaCharset_QuotedGT(t *testing.T) {
	t.Parallel()

	const iso = "iso-8859-1"
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "quoted-gt-before-charset",
			in:   `<meta data-x=">" charset="iso-8859-1">`,
			want: iso,
		},
		{
			name: "single-quoted-gt-before-charset",
			in:   `<meta data-x='>' charset='iso-8859-1'>`,
			want: iso,
		},
		{
			name: "quoted-gt-http-equiv",
			in:   `<meta data-x=">" http-equiv="Content-Type" content="text/html; charset=iso-8859-1">`,
			want: iso,
		},
		{
			name: "quoted-gt-in-preceding-tag",
			in:   `<a title=">"><meta charset="iso-8859-1">`,
			want: iso,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, extractMetaCharset([]byte(tc.in)))
		})
	}
}

// extractMetaCharset must mirror the main parser's char-data rule: a '<' begins
// markup ONLY when the byte after it is '/', '!', '?', or an ASCII letter. A
// literal non-tag '<' (e.g. `< " >` or `<x="...`) is ordinary character data and
// must be stepped over as a single byte WITHOUT entering the quote-aware tag-skip.
// Otherwise a stray '<' carrying a quote would put the scanner into quote state,
// make it ignore every later '>' (including the real tag terminator), and swallow
// a genuine <meta charset=...> that follows — missing the declaration.
// (Regression: PR #821 / HTML-101 final.)
func TestExtractMetaCharset_LiteralLessThan(t *testing.T) {
	t.Parallel()

	const iso = "iso-8859-1"
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "quote-bearing-non-tag-before-meta",
			in:   `< " ><meta charset=iso-8859-1>`,
			want: iso,
		},
		{
			name: "lt-equals-quote-before-meta",
			in:   `< x="><meta charset="iso-8859-1">`,
			want: iso,
		},
		{
			name: "lt-digit-before-meta",
			in:   `a < 3 && 4 > b <meta charset='iso-8859-1'>`,
			want: iso,
		},
		{
			name: "lt-space-before-meta",
			in:   `< not a tag <meta charset=iso-8859-1>`,
			want: iso,
		},
		{
			name: "trailing-lone-lt",
			in:   `<meta charset=iso-8859-1><`,
			want: iso,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, extractMetaCharset([]byte(tc.in)))
		})
	}
}

// extractMetaCharset must inspect ONLY the first 1024 raw bytes, and must do so
// without lowercasing the whole document. newParser calls this (via
// declaredCharsetIs{Latin1,UTF8}) ahead of utf8.Valid on every in-memory
// Parse([]byte), so folding the entire input just to read its head would
// allocate proportional to the document — a resource regression. The scan must
// be bounded to the ~1 KiB prescan window regardless of input size.
// (Regression: PR #821 / HTML-101 r13.) This test is intentionally NOT parallel
// so the allocation measurement runs without concurrent sibling tests.
func TestExtractMetaCharset_BoundedPrefix(t *testing.T) {
	t.Run("declaration past the 1024-byte window is ignored", func(t *testing.T) {
		// >1024 bytes of filler push the real meta past the prescan window.
		pad := strings.Repeat("x", 2000)
		doc := []byte("<!doctype html><html><head>" + pad + `<meta charset="iso-8859-1">`)
		require.Empty(t, extractMetaCharset(doc))
	})

	t.Run("declaration within the window is honored in a huge doc", func(t *testing.T) {
		body := strings.Repeat("a", 4<<20) // 4 MiB of trailing valid UTF-8
		doc := []byte(`<meta charset="iso-8859-1">` + body)
		require.Equal(t, "iso-8859-1", extractMetaCharset(doc))
	})

	t.Run("allocation is bounded to the window, not the input", func(t *testing.T) {
		const docSize = 8 << 20 // 8 MiB of valid UTF-8 trailing the meta
		doc := append([]byte(`<meta charset="iso-8859-1">`), bytes.Repeat([]byte("a"), docSize)...)

		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		got := extractMetaCharset(doc)
		runtime.ReadMemStats(&after)
		require.Equal(t, "iso-8859-1", got)

		allocated := after.TotalAlloc - before.TotalAlloc
		// Generous ceiling: a few KiB for the bounded lowercase copy + result.
		// Whole-document folding (the regression) would allocate >= docSize.
		require.Less(t, allocated, uint64(64<<10),
			"prescan allocated %d bytes; must stay bounded to the ~1 KiB window, not the %d-byte input", allocated, docSize)
	})
}

// extractMetaCharset must honor the WHATWG per-meta attribute rules: a declared
// encoding counts ONLY from a real `charset` attribute, or from a `content`
// attribute's charset when paired with `http-equiv="content-type"`. A `charset=`
// token in any other attribute (data-charset, name, a non-pragma content value)
// is ignored — trusting it would corrupt a valid UTF-8 document, since the eager
// encoding commit overrides utf8.Valid. (Regression: PR #821 / HTML-101.)
func TestExtractMetaCharset_AttributePrecision(t *testing.T) {
	t.Parallel()

	const iso = "iso-8859-1"
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		// Qualifying declarations.
		{name: "real-charset-attr", in: `<meta charset=iso-8859-1>`, want: iso},
		{name: "real-charset-attr-quoted", in: `<meta charset="iso-8859-1">`, want: iso},
		{
			name: "pragma-content-type",
			in:   `<meta http-equiv="content-type" content="text/html; charset=iso-8859-1">`,
			want: iso,
		},
		{
			name: "pragma-content-type-attr-order-swapped",
			in:   `<meta content="text/html; charset=iso-8859-1" http-equiv="content-type">`,
			want: iso,
		},
		// Non-qualifying: charset= sits in the wrong attribute.
		{name: "data-charset-ignored", in: `<meta data-charset=iso-8859-1>`, want: ""},
		{
			name: "non-pragma-content-ignored",
			in:   `<meta name=description content="charset=iso-8859-1">`,
			want: "",
		},
		{
			name: "content-without-http-equiv-ignored",
			in:   `<meta content="text/html; charset=iso-8859-1">`,
			want: "",
		},
		{
			name: "http-equiv-not-content-type-ignored",
			in:   `<meta http-equiv="refresh" content="charset=iso-8859-1">`,
			want: "",
		},
		{
			name: "name-attr-value-ignored",
			in:   `<meta name="charset=iso-8859-1">`,
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, extractMetaCharset([]byte(tc.in)))
		})
	}
}

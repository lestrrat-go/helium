package html_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// scanTokenCeiling mirrors the internal scanTokenLimit floor (defaultMaxContentSize,
// 16 MiB): the hard DoS ceiling for indivisible tag-level token scans. Unlike the
// content cap it does not shrink with a small MaxContentSize, so an over-cap case
// must exceed this fixed ceiling.
const scanTokenCeiling = 16 << 20

// TestScanRunOverCapFails verifies that the previously-unbounded PeekAt scans in
// the tag-level lexer (tag-name, attribute-name, intra-tag whitespace) cannot
// grow the cursor buffer without limit. A run exceeding the structural scan
// ceiling hard-fails with ErrContentSizeExceeded, the same surfaced error the
// comment/PI/attribute-value caps use, instead of buffering the whole run.
func TestScanRunOverCapFails(t *testing.T) {
	over := strings.Repeat("a", scanTokenCeiling+8)
	overWS := strings.Repeat(" ", scanTokenCeiling+8)

	testCases := []struct {
		name  string
		input string
	}{
		{name: "tag_name", input: "<" + over},
		{name: "end_tag_name", input: "<x></" + over},
		{name: "intra_tag_whitespace", input: "<p" + overWS + ">"},
		{name: "attr_name", input: "<p " + over + "=1>"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := html.NewParser().
				Parse(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap tag-level scan must fail with ErrContentSizeExceeded")
		})
	}
}

// TestScanTokensUnaffectedByTinyContentCap pins that the structural scan cap is
// independent of MaxContentSize: a caller setting a tiny content-chunking cap
// must still be able to parse ordinary multi-character tag names, attribute
// names, and intra-tag whitespace. (Binding these to MaxContentSize would reject
// names like "script" under MaxContentSize(1).)
func TestScanTokensUnaffectedByTinyContentCap(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{name: "raw_text_tag_name", input: "<script>x=1</script>"},
		{name: "rcdata_tag_name", input: "<title>hello</title>"},
		{name: "multichar_attr_name", input: `<p class="x">ok</p>`},
		{name: "intra_tag_whitespace", input: "<p     class=\"x\"     >ok</p>"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := html.NewParser().MaxContentSize(1).
				Parse(t.Context(), []byte(tc.input))
			require.NoError(t, err,
				"ordinary tag-level tokens must parse regardless of a tiny content cap")
		})
	}
}

// TestDoctypeQuotedLiteralOverCapFails verifies the DOCTYPE PUBLIC/SYSTEM
// literal scanner (parseQuotedString) bounds its previously-unbounded PeekAt
// scan: an unterminated, over-cap external/system identifier hard-fails with
// ErrContentSizeExceeded instead of buffering the whole literal until EOF.
func TestDoctypeQuotedLiteralOverCapFails(t *testing.T) {
	over := strings.Repeat("a", scanTokenCeiling+8)

	testCases := []struct {
		name  string
		input string
	}{
		// Unterminated SYSTEM literal (single quoted string).
		{name: "system", input: `<!DOCTYPE html SYSTEM "` + over},
		// Unterminated PUBLIC literal (first of two quoted strings).
		{name: "public", input: `<!DOCTYPE html PUBLIC "` + over},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := html.NewParser().Parse(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"unterminated over-cap DOCTYPE literal must fail with ErrContentSizeExceeded")
		})
	}
}

// TestDoctypeLiteralEmbeddedNULOverCapFails verifies that an embedded real NUL
// byte inside a DOCTYPE PUBLIC/SYSTEM literal does NOT terminate the scan early
// and bypass the hard cap. parseQuotedString must distinguish a genuine NUL
// content byte (HasByteAt true, PeekAt 0) from true end-of-input — like the
// comment/PI scanners — so an unterminated over-cap literal whose tail begins
// with a NUL still fails closed with ErrContentSizeExceeded rather than exiting
// the scanner early (leaving fatalErr unset) and emitting a partial subset.
func TestDoctypeLiteralEmbeddedNULOverCapFails(t *testing.T) {
	over := "\x00" + strings.Repeat("a", scanTokenCeiling+8)

	testCases := []struct {
		name  string
		input string
	}{
		// Unterminated SYSTEM literal with a leading NUL then a huge tail.
		{name: "system", input: `<!DOCTYPE html SYSTEM "` + over},
		// Unterminated PUBLIC literal (first of two quoted strings).
		{name: "public", input: `<!DOCTYPE html PUBLIC "` + over},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := html.NewParser().Parse(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap DOCTYPE literal with embedded NUL must fail with ErrContentSizeExceeded")
		})
	}
}

// TestOverCapEndTagNameDoesNotDrainStream verifies that an over-cap end-tag
// name surfaces its ErrContentSizeExceeded fatal PROMPTLY rather than first
// draining the remainder of an abusive stream in the "skip to '>'" loop. The
// trailing junk after the over-cap name must never be read. (countingReader and
// metaUTF8 are shared with rawtext_bounds_test.go.)
func TestOverCapEndTagNameDoesNotDrainStream(t *testing.T) {
	name := strings.Repeat("a", scanTokenCeiling+8)
	trailing := strings.Repeat("a", scanTokenCeiling)
	// A declared charset=utf-8 forces the streaming sanitize-reader path so the
	// parser pulls bytes on demand. Without it an undeclared valid-UTF-8 stream
	// buffers to EOF in the encoding layer and the read count could not
	// distinguish a prompt fail from a full drain.
	input := metaUTF8 + "<x></" + name + trailing
	r := &countingReader{data: []byte(input)}

	_, err := html.NewParser().ParseReader(t.Context(), r)
	require.ErrorIs(t, err, html.ErrContentSizeExceeded,
		"over-cap end-tag name must fail with ErrContentSizeExceeded")
	require.Less(t, r.pos, scanTokenCeiling+scanTokenCeiling/2,
		"over-cap end-tag name must not drain the trailing stream (read %d bytes)", r.pos)
}

// TestOverCapEndTagWhitespaceFailsClosed verifies that intra-tag whitespace
// AFTER an end-tag name is HARD-capped like every other intra-tag whitespace
// run, matching the documented contract on Parser.MaxContentSize /
// ErrContentSizeExceeded. The post-name whitespace in parseEndTag was previously
// drained by the unbounded "skip to '>'" loop, so `</p` + an over-cap whitespace
// run + `>` parsed successfully — an unbounded-scan DoS. It must now fail with
// ErrContentSizeExceeded. A normal end tag with a few trailing spaces still
// parses fine.
func TestOverCapEndTagWhitespaceFailsClosed(t *testing.T) {
	overWS := strings.Repeat(" ", scanTokenCeiling+8)

	t.Run("over_cap_fails", func(t *testing.T) {
		input := "<p>x</p" + overWS + ">"
		_, err := html.NewParser().Parse(t.Context(), []byte(input))
		require.ErrorIs(t, err, html.ErrContentSizeExceeded,
			"over-cap end-tag whitespace must fail with ErrContentSizeExceeded")
	})

	t.Run("normal_trailing_spaces_ok", func(t *testing.T) {
		_, err := html.NewParser().Parse(t.Context(), []byte("<p>x</p   >"))
		require.NoError(t, err,
			"an end tag with a few trailing spaces must still parse")
	})
}

// TestOverCapEndTagWhitespaceDoesNotDrainStream verifies that the over-cap
// end-tag whitespace fatal surfaces PROMPTLY on a streaming reader: parseEndTag
// must check fatalErr right after the bounded skipWhitespace and return BEFORE
// the "skip to '>'" drain loop. Otherwise the drain loop would keep reading
// (there is no '>' until far past the cap) and pull the entire abusive trailing
// stream. The trailing junk after the over-cap whitespace must never be read.
// (countingReader and metaUTF8 are shared with rawtext_bounds_test.go.)
func TestOverCapEndTagWhitespaceDoesNotDrainStream(t *testing.T) {
	overWS := strings.Repeat(" ", scanTokenCeiling+8)
	trailing := strings.Repeat("a", scanTokenCeiling)
	// A declared charset=utf-8 forces the streaming sanitize-reader path so the
	// parser pulls bytes on demand and the read count distinguishes a prompt
	// fail from a full drain of the trailing stream.
	input := metaUTF8 + "<p>x</p" + overWS + trailing
	r := &countingReader{data: []byte(input)}

	_, err := html.NewParser().ParseReader(t.Context(), r)
	require.ErrorIs(t, err, html.ErrContentSizeExceeded,
		"over-cap end-tag whitespace must fail with ErrContentSizeExceeded")
	require.Less(t, r.pos, scanTokenCeiling+scanTokenCeiling/2,
		"over-cap end-tag whitespace must not drain the trailing stream (read %d bytes)", r.pos)
}

// TestOverCapStartTagNameEmitsNoStrayText verifies that an over-cap start-tag
// name fails closed WITHOUT first publishing a stray '<' text node. A hard-cap
// failure must not produce partial SAX output before the fatal propagates.
func TestOverCapStartTagNameEmitsNoStrayText(t *testing.T) {
	over := strings.Repeat("a", scanTokenCeiling+8)

	var sawAngle bool
	sax := &html.SAXCallbacks{}
	sax.SetOnCharacters(html.CharactersFunc(func(ch []byte) error {
		if bytes.Contains(ch, []byte("<")) {
			sawAngle = true
		}
		return nil
	}))

	err := html.NewParser().ParseWithSAX(t.Context(), []byte("<"+over), sax)
	require.ErrorIs(t, err, html.ErrContentSizeExceeded,
		"over-cap start-tag name must fail with ErrContentSizeExceeded")
	require.False(t, sawAngle,
		"over-cap start-tag name must not emit a stray '<' text node")
}

// TestOverCapDoctypeWhitespaceDoesNotBlock is the regression for the parseDoctype
// streaming-block gap: an over-cap intra-DOCTYPE whitespace run sets fatalErr in
// the FIRST skipWhitespace, but parseDoctype must check fatalErr IMMEDIATELY and
// return rather than continuing into the next scanner (parseName / the second
// skipWhitespace), whose PeekAt would issue another (blocking) Read on a streaming
// reader stalled right at the over-cap boundary.
//
// The body is `<!DOCTYPE` followed by exactly an over-cap whitespace run and
// nothing else; once it is delivered the reader blocks forever on any further
// Read. Without the per-scanner fatal check parseDoctype's second skipWhitespace
// issues that blocking Read and the parse hangs (caught by the timeout); with the
// fix it surfaces ErrContentSizeExceeded promptly and never reads past the run.
// (blockOnExtraReadReader and metaUTF8 are shared with rawtext_bounds_test.go.)
func TestOverCapDoctypeWhitespaceDoesNotBlock(t *testing.T) {
	// scanTokenCeiling+8 whitespace bytes trip skipWhitespace's hard cap. A
	// <meta charset="utf-8"> forces the streaming sanitize path so the parser
	// pulls bytes on demand and the cursor refills incrementally toward the
	// boundary instead of buffering the whole stream up front.
	ws := strings.Repeat(" ", scanTokenCeiling+8)
	input := []byte(metaUTF8 + "<!DOCTYPE" + ws)
	r := &blockOnExtraReadReader{
		data:    input,
		maxRead: 1 << 16,
		block:   make(chan struct{}),
	}

	done := make(chan error, 1)
	go func() {
		_, err := html.NewParser().ParseReader(t.Context(), r)
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, html.ErrContentSizeExceeded,
			"over-cap DOCTYPE whitespace must fail with ErrContentSizeExceeded")
	case <-time.After(10 * time.Second):
		t.Fatal("parseDoctype issued a blocking Read after fatalErr was set (missing per-scanner fatal check)")
	}
}

// TestNonFatalInvalidStartTagStillEmitsText guards that the over-cap fail-fast
// in parseStartTag does NOT suppress the ordinary, non-fatal '<'-as-text
// fallback for a genuinely invalid start tag (e.g. '<' followed by whitespace).
func TestNonFatalInvalidStartTagStillEmitsText(t *testing.T) {
	var sawAngle bool
	sax := &html.SAXCallbacks{}
	sax.SetOnCharacters(html.CharactersFunc(func(ch []byte) error {
		if bytes.Contains(ch, []byte("<")) {
			sawAngle = true
		}
		return nil
	}))

	err := html.NewParser().ParseWithSAX(t.Context(), []byte("a < b"), sax)
	require.NoError(t, err)
	require.True(t, sawAngle,
		"a non-fatal lone '<' must still be emitted as text")
}

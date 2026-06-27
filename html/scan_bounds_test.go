package html_test

import (
	"bytes"
	"strings"
	"testing"

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

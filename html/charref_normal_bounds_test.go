package html_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestNormalDataOverCapNamedEntityFails is the regression for the normal
// character-data char-ref bypass (the sibling of the RCDATA bypass): a `&` in
// ordinary text followed by a huge alphanumeric run — e.g. `<div>&aaaa...(huge)`
// — must NOT be buffered whole while deciding whether it names an entity. The
// normal-data path now shares the bounded char-ref scanner, so a name longer
// than the longest known entity (31 chars) that matches no legacy prefix is
// LITERAL text charged against MaxContentSize; once it exceeds the cap before any
// terminator the parse fails with ErrContentSizeExceeded and no single chunk
// exceeds the cap. Before the fix the run was collected via an unbounded
// parseWhile, so peak memory grew with the input.
func TestNormalDataOverCapNamedEntityFails(t *testing.T) {
	const limit = 4
	const runLen = 4096 // far larger than any valid entity name or legacy prefix

	cases := []struct {
		name string
		body string
	}{
		{"unknown_no_semicolon", "&" + strings.Repeat("a", runLen)},
		{"unknown_semicolon", "&" + strings.Repeat("a", runLen) + ";"},
		// Begins with the legacy "amp" prefix but the long no-';' tail is
		// ambiguous until the run ends; over the cap it hard-fails too.
		{"legacy_prefix_long_tail", "&amp" + strings.Repeat("x", runLen)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := "<div>" + tc.body + "</div>"

			maxChunk := 0
			record := html.CharactersFunc(func(data []byte) error {
				if len(data) > maxChunk {
					maxChunk = len(data)
				}
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().MaxContentSize(limit).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap unresolved char-ref in normal character data must fail, not buffer the run")
			require.LessOrEqual(t, maxChunk, limit+16,
				"no single Characters chunk may exceed the cap before the abort")
		})
	}
}

// TestNormalDataCharRefResolution pins that ordinary char-ref resolution in
// normal character data is unchanged after routing through the bounded scanner:
// a known entity resolves, a legacy (HTML4) entity resolves without ';' and
// echoes its tail, an unknown name is echoed verbatim, and a numeric reference
// (including an overflow that normalizes to U+FFFD) resolves. Default cap.
func TestNormalDataCharRefResolution(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"amp_semicolon", "&amp;", "&"},
		{"lt_semicolon", "&lt;X", "<X"},
		{"legacy_amp_no_semicolon_tail", "&ampZ", "&Z"},
		{"unknown_semicolon", "&qqq;", "&qqq;"},
		{"numeric_dec", "&#65;", "A"},
		{"numeric_hex", "&#x41;", "A"},
		{"numeric_overflow", "&#9999999999;", "�"},
		// Longer than the longest known entity (31) yet tiny vs the default cap:
		// must be echoed literally, exactly like the previous unbounded path.
		{"long_unknown_preserved", "&" + strings.Repeat("a", 40), "&" + strings.Repeat("a", 40)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := "<div>" + tc.body + "</div>"

			var got strings.Builder
			record := html.CharactersFunc(func(data []byte) error {
				got.Write(data)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)
			sax.SetOnCDataBlock(html.CDataBlockFunc(record))

			err := html.NewParser().ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err)
			require.Equal(t, tc.want, got.String(),
				"char-ref in normal character data must resolve like the canonical path")
		})
	}
}

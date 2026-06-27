package html_test

import (
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

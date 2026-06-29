package html_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestAttrValueOverCapFails verifies that an attribute value exceeding
// MaxContentSize before its terminator hard-fails with ErrContentSizeExceeded
// instead of buffering without limit. Covers both quoted (no closing quote)
// and unquoted forms, matching the cap enforced on comment/PI/char-data.
func TestAttrValueOverCapFails(t *testing.T) {
	const limit = 64
	over := strings.Repeat("a", limit*4)

	testCases := []struct {
		name  string
		input string
	}{
		{name: "quoted_unterminated", input: `<p x="` + over},
		{name: "quoted_terminated", input: `<p x="` + over + `">`},
		{name: "unquoted", input: `<p x=` + over + `>`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := html.NewParser().MaxContentSize(limit).
				Parse(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap attribute value must fail with ErrContentSizeExceeded")
		})
	}
}

// TestAttrValueEntityRunOverCapFails verifies that a '&'-led or '&#'-led run in
// an attribute value cannot bypass the per-byte hard cap. The entity-name and
// numeric-digit scans are bounded (maxEntityNameLen), so an unterminated run
// longer than the cap falls through to the capped loop and hard-fails with
// ErrContentSizeExceeded instead of slurping unbounded into one allocation.
func TestAttrValueEntityRunOverCapFails(t *testing.T) {
	const limit = 64
	overName := strings.Repeat("a", limit*4)   // alphanumeric entity-name run
	overDigits := strings.Repeat("9", limit*4) // decimal numeric-ref run

	testCases := []struct {
		name  string
		input string
	}{
		{name: "named_run", input: `<p x="&` + overName},
		{name: "numeric_decimal_run", input: `<p x="&#` + overDigits},
		{name: "numeric_hex_run", input: `<p x="&#x` + overName},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := html.NewParser().MaxContentSize(limit).
				Parse(t.Context(), []byte(tc.input))
			require.ErrorIs(t, err, html.ErrContentSizeExceeded,
				"over-cap entity-led attribute run must fail with ErrContentSizeExceeded")
		})
	}
}

// TestAttrValueWithinCapParses verifies an attribute value at exactly the cap
// (followed by its terminator) parses cleanly, pinning the off-by-one boundary.
func TestAttrValueWithinCapParses(t *testing.T) {
	const limit = 64
	atCap := strings.Repeat("a", limit)

	testCases := []struct {
		name  string
		input string
	}{
		{name: "quoted", input: `<p x="` + atCap + `">ok</p>`},
		{name: "unquoted", input: `<p x=` + atCap + `>ok</p>`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := html.NewParser().MaxContentSize(limit).
				Parse(t.Context(), []byte(tc.input))
			require.NoError(t, err, "attribute value within cap must parse")
		})
	}
}

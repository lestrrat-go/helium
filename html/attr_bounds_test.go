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

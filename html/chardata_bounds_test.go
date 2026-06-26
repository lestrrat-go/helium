package html_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// TestNormalTextNamedRefBounded pins that a named character reference in the
// NORMAL data state (ordinary element content) is bounded by MaxContentSize the
// same way the RCDATA path is, instead of routing through the old unbounded
// parseCharRef scan. A long unresolved name that still fits the cap is echoed
// literally; one that exceeds the cap hard-fails with ErrContentSizeExceeded
// rather than buffering the whole run.
func TestNormalTextNamedRefBounded(t *testing.T) {
	t.Run("within_cap_preserved", func(t *testing.T) {
		const limit = 100
		// 40 chars: longer than the fixed 32-byte entity-name lookahead (so it
		// takes the saturated branch) yet well within MaxContentSize(100). It
		// resolves to no known entity or legacy prefix, so it is echoed verbatim.
		run := strings.Repeat("z", 40)
		body := "&" + run
		input := "<p>" + body + "</p>"

		var got strings.Builder
		record := html.CharactersFunc(func(data []byte) error {
			got.Write(data)
			return nil
		})
		sax := &html.SAXCallbacks{}
		sax.SetOnCharacters(record)

		err := html.NewParser().MaxContentSize(limit).
			ParseWithSAX(t.Context(), []byte(input), sax)
		require.NoError(t, err,
			"a within-cap named reference in normal text must parse")
		require.Equal(t, body, got.String(),
			"within-cap unresolved named run must be echoed literally")
	})

	t.Run("over_cap_fails", func(t *testing.T) {
		const limit = 8
		// "&" + a run far larger than the cap. Previously this buffered the whole
		// run via the unbounded parseWhile; now it hard-fails.
		body := "&" + strings.Repeat("z", limit+200)
		input := "<p>" + body + "</p>"

		_, err := html.NewParser().MaxContentSize(limit).
			Parse(t.Context(), []byte(input))
		require.ErrorIs(t, err, html.ErrContentSizeExceeded,
			"an over-cap named reference in normal text must hard-fail")
	})
}

// TestNormalTextNumericRefBounded pins that a numeric character reference in the
// normal data state consumes its digit run in bounded chunks (with overflow
// saturation) rather than materializing the whole run via the old unbounded
// parseWhile. An arbitrarily long overflowing run parses successfully and
// normalizes to U+FFFD, the same output the RCDATA path produces.
func TestNormalTextNumericRefBounded(t *testing.T) {
	const limit = 8

	for _, tc := range []struct {
		name string
		body string
	}{
		{"decimal", "&#" + strings.Repeat("9", 100000) + ";"},
		{"hex", "&#x" + strings.Repeat("f", 100000) + ";"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := "<p>" + tc.body + "</p>"

			var got strings.Builder
			record := html.CharactersFunc(func(data []byte) error {
				got.Write(data)
				return nil
			})
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(record)

			err := html.NewParser().MaxContentSize(limit).
				ParseWithSAX(t.Context(), []byte(input), sax)
			require.NoError(t, err,
				"a long overflowing numeric reference must parse via the bounded path")
			require.Equal(t, "�", got.String(),
				"an overflowing numeric reference normalizes to U+FFFD")
		})
	}
}

// TestStripBlanksLeadingSpaceBeforeMultibyteRune pins the clamp/whitespace-prefix
// fix: under StripBlanks(true) with a cap that falls inside the FIRST
// non-whitespace rune, the scanner must not flush a whitespace-only prefix that
// emitCharacters then suppresses, dropping the significant leading space. For
// "<p> é</p>" under MaxContentSize(1) the leading space rides along with the
// whole "é" rune, so the emitted text is " é".
func TestStripBlanksLeadingSpaceBeforeMultibyteRune(t *testing.T) {
	input := "<p> é</p>"

	var got strings.Builder
	record := html.CharactersFunc(func(data []byte) error {
		got.Write(data)
		return nil
	})
	sax := &html.SAXCallbacks{}
	sax.SetOnCharacters(record)

	err := html.NewParser().StripBlanks(true).MaxContentSize(1).
		ParseWithSAX(t.Context(), []byte(input), sax)
	require.NoError(t, err,
		"a significant run beginning with a multibyte rune must parse under StripBlanks")
	require.Equal(t, " é", got.String(),
		"the significant leading whitespace before the first non-whitespace rune must be preserved")
}

package encoding_test

import (
	"testing"

	xmlenc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/stretchr/testify/require"
)

// TestUnicodeBOMFamily verifies the canonical Unicode-family classification the
// BOM/encoding conflict check relies on. Every alias resolves through Load, so
// the punctuation-stripping normalizer and the full alias set are exercised
// without a parallel table.
func TestUnicodeBOMFamily(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		want string
	}{
		// UTF-8 and its aliases.
		{"UTF-8", "utf-8"},
		{"utf8", "utf-8"},
		{"unicode-1-1-utf-8", "utf-8"},
		{"unicode-2-0-utf-8", "utf-8"},

		// Endian-specific UTF-16 aliases.
		{"UTF-16BE", "utf-16be"},
		{"utf16be", "utf-16be"},
		{"unicodeFFFE", "utf-16be"},
		{"UTF-16LE", "utf-16le"},
		{"utf16le", "utf-16le"},
		{"unicodeFEFF", "utf-16le"},

		// Generic UTF-16 (endianness from the BOM) — compatible with either.
		{"UTF-16", "utf-16"},
		{"utf16", "utf-16"},
		{"unicode", "utf-16"},
		{"csUnicode", "utf-16"},

		// Non-UTF-16/UTF-8 encodings and unknown names classify as "".
		{"ISO-8859-1", ""},
		{"US-ASCII", ""},
		{"UTF-32", ""},
		{"windows-1252", ""},
		{"no-such-encoding", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, xmlenc.UnicodeBOMFamily(tc.name))
		})
	}
}

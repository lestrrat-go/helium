package xmlbase64_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/xmlbase64"
	"github.com/stretchr/testify/require"
)

// decoderBufferBytes is what encoding/base64 allocates for a value of chars
// characters: it sizes the output buffer from the character count alone and
// allocates it before validating anything, so this is the cost a rejected
// decode still pays.
func decoderBufferBytes(chars int) int {
	return chars / 4 * 3
}

func TestDecodeString(t *testing.T) {
	t.Run("accepts XML whitespace between characters", func(t *testing.T) {
		decoded, err := xmlbase64.DecodeString(" QUJD\n\tREVG ")
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
	})

	t.Run("rejects non-whitespace junk", func(t *testing.T) {
		_, err := xmlbase64.DecodeString("!!!not-base64!!!")
		require.Error(t, err)
	})
}

// TestDecodedLen covers the two halves of DecodedLen's contract: an exact
// count for input DecodeString accepts, and — for input it rejects — a count
// that never falls below the buffer the rejected decode allocates. The second
// half is what lets a caller weigh a value against a byte budget: a count that
// trusted a lexical form the decoder refuses would report less than the decode
// costs.
func TestDecodedLen(t *testing.T) {
	t.Run("exact for accepted input", func(t *testing.T) {
		for n := range 512 {
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i)
			}
			encoded := base64.StdEncoding.EncodeToString(data)
			require.Equal(t, n, xmlbase64.DecodedLen(encoded), "plain, n=%d", n)

			// The same value line-wrapped and indented, as real XML
			// Signature/Encryption producers emit it.
			wrapped := wrapWithWhitespace(encoded)
			decoded, err := xmlbase64.DecodeString(wrapped)
			require.NoError(t, err, "wrapped, n=%d", n)
			require.Len(t, decoded, n, "wrapped, n=%d", n)
			require.Equal(t, n, xmlbase64.DecodedLen(wrapped), "wrapped, n=%d", n)
		}
	})

	t.Run("whitespace-heavy accepted value counts exactly", func(t *testing.T) {
		// 3 bytes of payload buried in whitespace: the count must follow the
		// characters, not the string length.
		const s = "  \t\r\n Q U J \n D \t\r\n  "
		decoded, err := xmlbase64.DecodeString(s)
		require.NoError(t, err)
		require.Equal(t, []byte("ABC"), decoded)
		require.Equal(t, 3, xmlbase64.DecodedLen(s))
	})

	// Every value here is rejected by DecodeString, so there is no decoded
	// length to be exact about. What must hold is that the count covers the
	// decoder's buffer, which it allocates before rejecting.
	t.Run("rejected input is never under-counted", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			value string
			chars int
		}{
			// All padding. Deducting one byte per '=' would report 0.
			{name: "all padding", value: strings.Repeat("=", 4096), chars: 4096},
			// Padding mid-value. Every quantum is 2 of 4 padding, so
			// deducting per '=' would report half the decoder's buffer.
			{name: "repeated AA==", value: strings.Repeat("AA==", 1024), chars: 4096},
			// Padding ahead of the final quantum.
			{name: "padding before data", value: "==AAAAAA", chars: 8},
			// Three trailing '=' is one too many for any encoding.
			{name: "over-long padding", value: "A===", chars: 4},
			// Junk outside the alphabet: no padding to mis-deduct, but the
			// buffer is still allocated.
			{name: "outside the alphabet", value: strings.Repeat("!", 4096), chars: 4096},
			// Padding split by whitespace is still padding, and still not
			// at the end.
			{name: "whitespace between padding", value: "Q U = \n = B A = =", chars: 8},
			// Well-formed trailing padding on a body that is NOT base64.
			// The shape alone says two bytes may be deducted; the body
			// says the decoder will refuse the value and allocate the
			// whole quantum anyway.
			{name: "junk body with trailing padding", value: "!!==", chars: 4},
			// The same, with one alphabet character ahead of the junk.
			{name: "part-alphabet body with trailing padding", value: "A!==", chars: 4},
			// And the same value line-wrapped, the way an XML producer
			// would emit it.
			{name: "junk body with padding and whitespace", value: "! \t!\r\n==", chars: 4},
			// One '=' rather than two, so the deduction is smaller but
			// just as wrong.
			{name: "junk body with single trailing padding", value: "!!!=", chars: 4},
			// More than one quantum, so the shape check is not a
			// length-four special case.
			{name: "junk body with trailing padding, many quanta", value: strings.Repeat("!", 4094) + "==", chars: 4096},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := xmlbase64.DecodeString(tc.value)
				require.Error(t, err, "value must be rejected for this case to mean anything")
				require.GreaterOrEqual(t, xmlbase64.DecodedLen(tc.value), decoderBufferBytes(tc.chars))
			})
		}
	})

	t.Run("empty value counts zero", func(t *testing.T) {
		require.Equal(t, 0, xmlbase64.DecodedLen(""))
		require.Equal(t, 0, xmlbase64.DecodedLen(" \t\r\n"))
	})

	t.Run("never negative", func(t *testing.T) {
		for _, s := range []string{"=", "==", "===", "====", "A=", "==="} {
			require.GreaterOrEqual(t, xmlbase64.DecodedLen(s), 0, "value=%q", s)
		}
	})

	// Both halves of the contract at once, over every string the alphabet
	// below can build. The alphabet carries one character of each kind that
	// steers DecodedLen — two alphabet characters, padding, two whitespace
	// forms, and two characters outside the alphabet — so the enumeration
	// reaches every lexical shape the counter distinguishes rather than the
	// shapes anyone thought to name.
	t.Run("exhaustive over short values", func(t *testing.T) {
		const alphabet = "AB= \n!@"
		for _, s := range enumerate(alphabet, 6) {
			count := xmlbase64.DecodedLen(s)
			decoded, err := xmlbase64.DecodeString(s)
			if err != nil {
				require.GreaterOrEqual(t, count, decoderBufferBytes(strippedLen(s)), "rejected value=%q", s)
				continue
			}
			require.Equal(t, len(decoded), count, "accepted value=%q", s)
		}
	})
}

// enumerate returns every string of length 0 through maxLen over alphabet.
func enumerate(alphabet string, maxLen int) []string {
	out := []string{""}
	prev := []string{""}
	for range maxLen {
		next := make([]string, 0, len(prev)*len(alphabet))
		for _, s := range prev {
			for i := range len(alphabet) {
				next = append(next, s+string(alphabet[i]))
			}
		}
		out = append(out, next...)
		prev = next
	}
	return out
}

// strippedLen counts the characters left after DecodeString strips XML
// whitespace, which is what encoding/base64 sizes its output buffer from.
func strippedLen(s string) int {
	var n int
	for i := range len(s) {
		switch s[i] {
		case ' ', '\t', '\r', '\n':
		default:
			n++
		}
	}
	return n
}

// wrapWithWhitespace line-wraps s at 16 characters and indents each line,
// producing the interspersed XML whitespace xs:base64Binary permits.
func wrapWithWhitespace(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += 16 {
		end := min(i+16, len(s))
		b.WriteString("\n\t  ")
		b.WriteString(s[i:end])
	}
	b.WriteString("\n  ")
	return b.String()
}

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
// trusted malformed padding would report far less than the decode costs.
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

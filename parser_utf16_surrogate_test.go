package helium_test

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// utf16beWithUnits encodes s as UTF-16BE (with a BOM) and appends any extra raw
// 16-bit code units verbatim, so malformed sequences (e.g. an unpaired
// surrogate) can be injected. The pieces are concatenated in order.
func utf16beDoc(parts ...any) []byte {
	out := []byte{0xFE, 0xFF} // UTF-16BE BOM
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			for _, u := range utf16.Encode([]rune(v)) {
				out = binary.BigEndian.AppendUint16(out, u)
			}
		case uint16:
			out = binary.BigEndian.AppendUint16(out, v)
		default:
			panic("unsupported part type")
		}
	}
	return out
}

func TestParseUTF16UnpairedSurrogate(t *testing.T) {
	const unpairedHigh = uint16(0xD800) // lone high surrogate: malformed input

	t.Run("unpaired surrogate in text content is rejected", func(t *testing.T) {
		doc := utf16beDoc("<r>", unpairedHigh, "</r>")
		_, err := helium.NewParser().Parse(t.Context(), doc)
		require.Error(t, err, "unpaired surrogate in text content must be a fatal error")
	})

	t.Run("unpaired surrogate in attribute value is rejected", func(t *testing.T) {
		doc := utf16beDoc(`<r a="`, unpairedHigh, `"/>`)
		_, err := helium.NewParser().Parse(t.Context(), doc)
		require.Error(t, err, "unpaired surrogate in attribute value must be a fatal error")
	})

	t.Run("genuine U+FFFD in text content is accepted", func(t *testing.T) {
		doc := utf16beDoc("<r>�</r>")
		parsed, err := helium.NewParser().Parse(t.Context(), doc)
		require.NoError(t, err, "genuine U+FFFD is a valid XML character")
		root := parsed.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "�", string(root.Content()))
	})

	t.Run("genuine U+FFFD in attribute value is accepted", func(t *testing.T) {
		doc := utf16beDoc("<r a=\"�\"/>")
		parsed, err := helium.NewParser().Parse(t.Context(), doc)
		require.NoError(t, err, "genuine U+FFFD is a valid XML character")
		require.NotNil(t, parsed.DocumentElement())
	})
}

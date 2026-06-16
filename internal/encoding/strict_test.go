package encoding_test

import (
	"testing"

	xmlenc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/stretchr/testify/require"
)

func TestStrictDecodeUTF16(t *testing.T) {
	t.Parallel()

	t.Run("genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16be")
		require.NotNil(t, e)
		// U+FFFD encoded as a real UTF-16BE code unit: 0xFFFD.
		got, err := e.NewDecoder().Bytes([]byte{0x00, 0x41, 0xFF, 0xFD, 0x00, 0x42})
		require.NoError(t, err)
		require.Equal(t, "A�B", string(got))
	})

	t.Run("unpaired high surrogate rejected", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16be")
		require.NotNil(t, e)
		// 0xD800 is an unpaired high surrogate — malformed input.
		_, err := e.NewDecoder().Bytes([]byte{0x00, 0x41, 0xD8, 0x00, 0x00, 0x42})
		require.Error(t, err)
	})

	t.Run("trailing odd byte rejected", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16be")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0x00, 0x41, 0x00})
		require.Error(t, err)
	})

	t.Run("little-endian genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16le")
		require.NotNil(t, e)
		got, err := e.NewDecoder().Bytes([]byte{0x41, 0x00, 0xFD, 0xFF, 0x42, 0x00})
		require.NoError(t, err)
		require.Equal(t, "A�B", string(got))
	})

	t.Run("little-endian unpaired surrogate rejected", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16le")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0x41, 0x00, 0x00, 0xD8, 0x42, 0x00})
		require.Error(t, err)
	})
}

func TestStrictDecodeUTF16Plain(t *testing.T) {
	t.Parallel()

	// A normal ASCII-only UTF-16 document still decodes cleanly.
	e := xmlenc.Load("utf-16be")
	require.NotNil(t, e)
	got, err := e.NewDecoder().Bytes([]byte{0x00, 0x68, 0x00, 0x69})
	require.NoError(t, err)
	require.Equal(t, "hi", string(got))
}

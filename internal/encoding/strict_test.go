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

	t.Run("lone low surrogate rejected (BE)", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16be")
		require.NotNil(t, e)
		// 0xDC00 is a lone low surrogate with no preceding high surrogate.
		_, err := e.NewDecoder().Bytes([]byte{0x00, 0x41, 0xDC, 0x00})
		require.Error(t, err)
	})

	t.Run("lone low surrogate rejected (LE)", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16le")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0x41, 0x00, 0x00, 0xDC})
		require.Error(t, err)
	})

	t.Run("two unpaired surrogates rejected (ErrShortDst trigger)", func(t *testing.T) {
		t.Parallel()

		// Round-4 finding 1: two consecutive unpaired high surrogates that the
		// base decoder may emit before filling dst. Must still be rejected.
		e := xmlenc.Load("utf-16be")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0xD8, 0x00, 0xD8, 0x00, 0x00, 0x41})
		require.Error(t, err)
	})

	t.Run("valid surrogate pair accepted (BE)", func(t *testing.T) {
		t.Parallel()

		// U+1F600 = high D83D, low DE00.
		e := xmlenc.Load("utf-16be")
		require.NotNil(t, e)
		got, err := e.NewDecoder().Bytes([]byte{0xD8, 0x3D, 0xDE, 0x00})
		require.NoError(t, err)
		require.Equal(t, "\U0001F600", string(got))
	})

	t.Run("valid surrogate pair accepted (LE)", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16le")
		require.NotNil(t, e)
		got, err := e.NewDecoder().Bytes([]byte{0x3D, 0xD8, 0x00, 0xDE})
		require.NoError(t, err)
		require.Equal(t, "\U0001F600", string(got))
	})
}

func TestStrictDecodeUTF16BOM(t *testing.T) {
	t.Parallel()

	t.Run("BOM-BE genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16")
		require.NotNil(t, e)
		// BE BOM + "A" + real U+FFFD + "B".
		got, err := e.NewDecoder().Bytes([]byte{0xFE, 0xFF, 0x00, 0x41, 0xFF, 0xFD, 0x00, 0x42})
		require.NoError(t, err)
		require.Equal(t, "A�B", string(got))
	})

	t.Run("BOM-LE genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16")
		require.NotNil(t, e)
		// LE BOM + "A" + real U+FFFD + "B".
		got, err := e.NewDecoder().Bytes([]byte{0xFF, 0xFE, 0x41, 0x00, 0xFD, 0xFF, 0x42, 0x00})
		require.NoError(t, err)
		require.Equal(t, "A�B", string(got))
	})

	t.Run("BOM-BE unpaired surrogate rejected", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16")
		require.NotNil(t, e)
		// BE BOM + "A" + unpaired high surrogate + "B".
		_, err := e.NewDecoder().Bytes([]byte{0xFE, 0xFF, 0x00, 0x41, 0xD8, 0x00, 0x00, 0x42})
		require.Error(t, err)
	})

	t.Run("BOM-LE unpaired surrogate rejected", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-16")
		require.NotNil(t, e)
		// LE BOM + "A" + unpaired high surrogate + "B".
		_, err := e.NewDecoder().Bytes([]byte{0xFF, 0xFE, 0x41, 0x00, 0x00, 0xD8, 0x42, 0x00})
		require.Error(t, err)
	})
}

func TestStrictDecodeUTF32(t *testing.T) {
	t.Parallel()

	t.Run("genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-32be")
		require.NotNil(t, e)
		got, err := e.NewDecoder().Bytes([]byte{0x00, 0x00, 0x00, 0x41, 0x00, 0x00, 0xFF, 0xFD})
		require.NoError(t, err)
		require.Equal(t, "A�", string(got))
	})

	t.Run("BOM genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-32")
		require.NotNil(t, e)
		// BE BOM + "A" + real U+FFFD.
		got, err := e.NewDecoder().Bytes([]byte{0x00, 0x00, 0xFE, 0xFF, 0x00, 0x00, 0x00, 0x41, 0x00, 0x00, 0xFF, 0xFD})
		require.NoError(t, err)
		require.Equal(t, "A�", string(got))
	})

	t.Run("out-of-range scalar rejected", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-32be")
		require.NotNil(t, e)
		// 0x00110000 is beyond the Unicode range — malformed input.
		_, err := e.NewDecoder().Bytes([]byte{0x00, 0x11, 0x00, 0x00})
		require.Error(t, err)
	})

	t.Run("BE out-of-range scalar that is FFFD in LE rejected", func(t *testing.T) {
		t.Parallel()

		// Round-4 finding 2: FD FF 00 00 read big-endian is 0xFDFF0000, far out
		// of range — malformed. The old "either byte order" check read it as
		// 0xFFFD little-endian and wrongly accepted it.
		e := xmlenc.Load("utf-32be")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0xFD, 0xFF, 0x00, 0x00})
		require.Error(t, err)
	})

	t.Run("LE genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-32le")
		require.NotNil(t, e)
		got, err := e.NewDecoder().Bytes([]byte{0x41, 0x00, 0x00, 0x00, 0xFD, 0xFF, 0x00, 0x00})
		require.NoError(t, err)
		require.Equal(t, "A�", string(got))
	})

	t.Run("BOM-LE genuine U+FFFD accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-32")
		require.NotNil(t, e)
		// LE BOM (FF FE 00 00) + "A" + real U+FFFD.
		got, err := e.NewDecoder().Bytes([]byte{0xFF, 0xFE, 0x00, 0x00, 0x41, 0x00, 0x00, 0x00, 0xFD, 0xFF, 0x00, 0x00})
		require.NoError(t, err)
		require.Equal(t, "A�", string(got))
	})

	t.Run("astral char accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("utf-32be")
		require.NotNil(t, e)
		// U+1F600.
		got, err := e.NewDecoder().Bytes([]byte{0x00, 0x01, 0xF6, 0x00})
		require.NoError(t, err)
		require.Equal(t, "\U0001F600", string(got))
	})

	t.Run("surrogate scalar rejected", func(t *testing.T) {
		t.Parallel()

		// A surrogate code point (0xD800) is not a valid UTF-32 scalar.
		e := xmlenc.Load("utf-32be")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0x00, 0x00, 0xD8, 0x00})
		require.Error(t, err)
	})
}

func TestStrictDecodeUCS4Swap(t *testing.T) {
	t.Parallel()

	t.Run("trailing partial unit rejected", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("ucs4_2143")
		require.NotNil(t, e)
		// "A" in 2143 order is 00 00 41 00; a trailing 0xff is an incomplete unit.
		_, err := e.NewDecoder().Bytes([]byte{0x00, 0x00, 0x41, 0x00, 0xff})
		require.Error(t, err)
	})

	t.Run("trailing partial unit rejected (3412)", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("ucs4_3412")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0x41, 0x00, 0x00, 0x00, 0xff, 0xff})
		require.Error(t, err)
	})

	t.Run("complete units accepted", func(t *testing.T) {
		t.Parallel()

		e := xmlenc.Load("ucs4_2143")
		require.NotNil(t, e)
		// "A" in 2143 byte order: code point 0x41 => big-endian 00 00 00 41,
		// 2143 swap of (b0 b1 b2 b3)->(b1 b0 b3 b2) gives 00 00 41 00.
		got, err := e.NewDecoder().Bytes([]byte{0x00, 0x00, 0x41, 0x00})
		require.NoError(t, err)
		require.Equal(t, "A", string(got))
	})

	t.Run("genuine U+FFFD accepted (2143)", func(t *testing.T) {
		t.Parallel()

		// Round-4 finding 2: genuine U+FFFD in UCS-4-2143. Code point 0xFFFD =>
		// big-endian 00 00 FF FD; 2143 swap (b1 b0 b3 b2) gives 00 00 FD FF.
		// The old "either byte order" check read this big-endian as 0xFFFD and
		// wrongly rejected it (mismatch vs decoder output).
		e := xmlenc.Load("ucs4_2143")
		require.NotNil(t, e)
		got, err := e.NewDecoder().Bytes([]byte{0x00, 0x00, 0xFD, 0xFF})
		require.NoError(t, err)
		require.Equal(t, "�", string(got))
	})

	t.Run("out-of-range scalar rejected (2143)", func(t *testing.T) {
		t.Parallel()

		// 0x00110000 big-endian => 00 11 00 00; 2143 swap (b1 b0 b3 b2) = 11 00 00 00.
		e := xmlenc.Load("ucs4_2143")
		require.NotNil(t, e)
		_, err := e.NewDecoder().Bytes([]byte{0x11, 0x00, 0x00, 0x00})
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

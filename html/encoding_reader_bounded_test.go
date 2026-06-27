package html

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// countingASCIIReader yields pure-ASCII bytes ('a') up to remaining, recording
// the total number of bytes it has actually handed out. It models a long (here
// finite, to keep the test bounded even against a buggy whole-buffer code path)
// undeclared-charset stream that is valid UTF-8 throughout.
type countingASCIIReader struct {
	remaining int
	total     int
}

func (r *countingASCIIReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := min(len(p), r.remaining)
	for i := range n {
		p[i] = 'a'
	}
	r.remaining -= n
	r.total += n
	return n, nil
}

// TestDeferredLatin1ReaderBoundedBufferingFailsClosed guards the memory bound:
// an undeclared all-valid-UTF-8 (ASCII) stream that never reveals its encoding
// must not be buffered without limit. Once deferredLatin1MaxBuffer bytes have
// been seen with no non-UTF-8 byte and no EOF, the exact UTF-8-vs-Latin-1
// decision still cannot be made, so the reader FAILS CLOSED with a bounded-input
// error (ErrContentSizeExceeded) instead of committing to UTF-8 (which could
// silently mis-decode a later high byte and diverge from Parse([]byte)). Memory
// stays bounded: the reader stops reading near the cap, not at the 32 MiB end.
func TestDeferredLatin1ReaderBoundedBufferingFailsClosed(t *testing.T) {
	t.Parallel()

	const sourceSize = 32 << 20 // 32 MiB of pure ASCII
	src := &countingASCIIReader{remaining: sourceSize}
	dr := newDeferredLatin1Reader(src)

	buf := make([]byte, 8192)
	n, err := dr.Read(buf)
	require.ErrorIs(t, err, ErrContentSizeExceeded,
		"an undeclared stream that stays valid UTF-8 past the cap must fail closed")
	require.Zero(t, n, "no irreversible output may be emitted on the fail-closed path")
	require.LessOrEqual(t, src.total, deferredLatin1MaxBuffer+8192,
		"the reader must stop near the cap, not buffer the whole stream")
}

// TestDeferredLatin1ReaderUnderCapDeliversAllUTF8 verifies the legitimate case:
// an undeclared all-ASCII stream that ends (EOF) below the buffering cap settles
// as UTF-8 and streams every byte through unchanged, never switching to Latin-1.
func TestDeferredLatin1ReaderUnderCapDeliversAllUTF8(t *testing.T) {
	t.Parallel()

	const sourceSize = deferredLatin1MaxBuffer / 2 // settles at EOF below the cap
	src := &countingASCIIReader{remaining: sourceSize}
	dr := newDeferredLatin1Reader(src)

	out, err := io.ReadAll(dr)
	require.NoError(t, err)
	require.Len(t, out, sourceSize, "every input byte must be delivered exactly once")
	for _, b := range out {
		require.EqualValues(t, 'a', b)
	}
	require.Empty(t, dr.detectedEncoding(), "an all-UTF-8 stream must not switch to Latin-1")
}

// TestDeferredLatin1ReaderOverCapThenHighByteErrors covers the pathological case
// the buffering cap exists for: an undeclared stream that stays valid UTF-8 past
// the cap and THEN contains a raw non-UTF-8 byte. The []byte path would
// reinterpret the WHOLE document as Latin-1, so the streaming reader cannot
// safely commit to UTF-8. It must fail closed with the bounded-input error
// BEFORE the late high byte is even read — never mis-decode it (verbatim or
// sanitized) into SAX/DOM.
func TestDeferredLatin1ReaderOverCapThenHighByteErrors(t *testing.T) {
	t.Parallel()

	const prefixSize = deferredLatin1MaxBuffer + 4096 // past the buffering cap
	var src bytes.Buffer
	src.Grow(prefixSize + 8)
	for range prefixSize {
		src.WriteByte('a')
	}
	src.WriteByte(0x93) // lone Windows-1252 byte: invalid UTF-8
	src.WriteByte('z')

	dr := newDeferredLatin1Reader(bytes.NewReader(src.Bytes()))

	out, err := io.ReadAll(dr)
	require.ErrorIs(t, err, ErrContentSizeExceeded,
		"an over-cap undeclared stream must fail closed, not mis-decode the late high byte")
	require.NotContains(t, out, byte(0x93), "the raw invalid byte must never leak into output")
	require.NotContains(t, string(out), "�",
		"no sanitized U+FFFD: the stream is rejected, not silently rewritten")
}

// TestDeferredLatin1ReaderInvalidByteAtCapBoundary covers the boundary case
// where a GENUINE non-UTF-8 byte lands exactly at the buffering cap:
// deferredLatin1MaxBuffer-1 ASCII bytes followed by a single invalid byte (a
// lone Windows-1252 byte such as 0x93 or 0x80). Such a byte is a "full"
// RuneError of size 1, not a truncated trailing rune, so the reader must
// reinterpret the WHOLE buffer as Windows-1252 (matching the in-memory []byte
// path) — the switch fires while scanning the buffer, before the cap's
// fail-closed check, so a real Latin-1 document whose first high byte sits right
// at the cap still decodes correctly rather than being rejected.
func TestDeferredLatin1ReaderInvalidByteAtCapBoundary(t *testing.T) {
	t.Parallel()

	for _, bad := range []byte{0x93, 0x80} {
		t.Run(fmt.Sprintf("byte_%#02x", bad), func(t *testing.T) {
			t.Parallel()

			src := make([]byte, deferredLatin1MaxBuffer)
			for i := range src {
				src[i] = 'a'
			}
			src[len(src)-1] = bad // invalid byte exactly at the cap boundary

			dr := newDeferredLatin1Reader(bytes.NewReader(src))
			out, err := io.ReadAll(dr)
			require.NoError(t, err, "a genuine high byte at the cap settles the encoding, not a fail-close")
			require.NotContains(t, out, bad, "raw invalid byte must never leak as UTF-8")
			require.True(t, utf8.Valid(out), "output must be well-formed UTF-8")
			require.Equal(t, encWindows1252, dr.detectedEncoding(),
				"an invalid byte at the cap boundary must flip the whole buffer to Windows-1252")
			require.Equal(t, latin1ToUTF8(src), out,
				"the whole buffer must be reinterpreted as Windows-1252")
		})
	}
}

// dataErrReader returns a scripted prefix of ASCII bytes, then on a single later
// Read delivers a final chunk of converted-able bytes together with a non-EOF
// error (which io.Reader explicitly permits), then keeps reporting the error.
type dataErrReader struct {
	prefix []byte
	tail   []byte
	err    error
	sent   bool
}

func (r *dataErrReader) Read(p []byte) (int, error) {
	if len(r.prefix) > 0 {
		n := copy(p, r.prefix)
		r.prefix = r.prefix[n:]
		return n, nil
	}
	if !r.sent {
		r.sent = true
		n := copy(p, r.tail)
		return n, r.err
	}
	return 0, r.err
}

// TestLatin1ReaderDrainsBytesBeforeError pins the latin1Reader sticky-drain
// ordering used by the declared charset=iso-8859-1 path. The underlying reader
// returns a final chunk of bytes together with a non-EOF error on the same Read.
// With a read buffer SMALLER than the converted chunk, every converted byte must
// still drain before the error surfaces — the error must never truncate buffered
// output (the bug was surfacing the error together with, or ahead of, the bytes).
func TestLatin1ReaderDrainsBytesBeforeError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("checksum mismatch")

	// Latin-1 high bytes (0xE9 = 'é') so each input byte converts to a 2-byte
	// UTF-8 rune, guaranteeing converted output larger than the read buffer.
	tail := bytes.Repeat([]byte{0xE9}, 32)
	lr := &latin1Reader{
		r:   &dataErrReader{prefix: []byte("ab"), tail: tail, err: sentinel},
		enc: encISO88591,
	}

	buf := make([]byte, 5)
	var out []byte
	var readErr error
	for {
		n, err := lr.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			readErr = err
			break
		}
	}

	require.ErrorIs(t, readErr, sentinel, "the non-EOF read error must surface")
	want := append([]byte("ab"), bytes.Repeat([]byte("é"), 32)...)
	require.Equal(t, want, out,
		"every converted byte must drain before the error truncates the stream")
}

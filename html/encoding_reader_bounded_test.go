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
// finite, to keep the test bounded even against the buggy whole-buffer code)
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

// TestDeferredLatin1ReaderBoundedBuffering guards against unbounded buffering:
// an all-valid-UTF-8 (ASCII) stream must reach an encoding decision and start
// emitting after a bounded prefix instead of being buffered whole until EOF.
// Buffering the entire stream defeats streaming and the parser's content caps —
// an unbounded-memory DoS for an endless ASCII source.
func TestDeferredLatin1ReaderBoundedBuffering(t *testing.T) {
	t.Parallel()

	const sourceSize = 32 << 20 // 32 MiB of pure ASCII
	src := &countingASCIIReader{remaining: sourceSize}
	dr := newDeferredLatin1Reader(src)

	buf := make([]byte, 8192)
	n, err := dr.Read(buf)
	require.NoError(t, err)
	require.Positive(t, n, "must emit output without first consuming the whole stream")
	require.Less(t, src.total, 8<<20,
		"must commit to a decision after a bounded prefix, not buffer the whole stream")
	for _, b := range buf[:n] {
		require.EqualValues(t, 'a', b)
	}
}

// TestDeferredLatin1ReaderCommittedPassthroughDeliversAll verifies that once the
// reader commits to UTF-8 after the bounded prefix it still streams the entire
// remaining input through unchanged (no bytes dropped or duplicated) and never
// falsely switches to Latin-1 for an all-UTF-8 stream.
func TestDeferredLatin1ReaderCommittedPassthroughDeliversAll(t *testing.T) {
	t.Parallel()

	const sourceSize = 4 << 20 // larger than the internal cap to force a commit
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

// TestDeferredLatin1ReaderSanitizesPostCommitInvalidByte covers the pathological
// case the bounded-buffer commit creates: an undeclared stream that stays valid
// UTF-8 for more than the cap and THEN contains a raw non-UTF-8 byte. After
// committing to UTF-8 the reader must NOT leak the raw invalid byte into the
// output; it sanitizes it to U+FFFD (matching the parser's decode-error handling)
// so SAX/DOM never sees ill-formed UTF-8. It must also not hang or OOM.
func TestDeferredLatin1ReaderSanitizesPostCommitInvalidByte(t *testing.T) {
	t.Parallel()

	const prefixSize = deferredLatin1MaxBuffer + 4096 // past the commit cap
	var src bytes.Buffer
	src.Grow(prefixSize + 8)
	for range prefixSize {
		src.WriteByte('a')
	}
	src.WriteByte(0x93) // lone Windows-1252 byte: invalid UTF-8
	src.WriteByte('z')

	dr := newDeferredLatin1Reader(bytes.NewReader(src.Bytes()))

	out, err := io.ReadAll(dr)
	require.NoError(t, err)
	require.NotContains(t, out, byte(0x93), "raw invalid byte must never leak into output")
	require.Contains(t, string(out), "�", "invalid byte must be replaced with U+FFFD")
	// Output = prefix + U+FFFD (3 bytes) + 'z'.
	require.Len(t, out, prefixSize+3+1, "all valid bytes delivered, invalid byte sanitized")
	require.Equal(t, byte('z'), out[len(out)-1], "bytes after the invalid one still flow through")
}

// TestDeferredLatin1ReaderInvalidByteAtCommitBoundary covers the boundary case
// where a GENUINE non-UTF-8 byte lands exactly at the bounded-buffer commit
// point: deferredLatin1MaxBuffer-1 ASCII bytes followed by a single invalid
// byte (a lone Windows-1252 byte such as 0x93 or 0x80). Such a byte is a "full"
// RuneError of size 1, not a truncated trailing rune, so the reader must
// reinterpret the WHOLE buffer as Windows-1252 (matching the in-memory []byte
// path) rather than mistaking it for an incomplete rune, committing to UTF-8,
// and flushing the raw invalid byte verbatim into SAX/DOM.
func TestDeferredLatin1ReaderInvalidByteAtCommitBoundary(t *testing.T) {
	t.Parallel()

	for _, bad := range []byte{0x93, 0x80} {
		t.Run(fmt.Sprintf("byte_%#02x", bad), func(t *testing.T) {
			t.Parallel()

			src := make([]byte, deferredLatin1MaxBuffer)
			for i := range src {
				src[i] = 'a'
			}
			src[len(src)-1] = bad // invalid byte exactly at the commit boundary

			dr := newDeferredLatin1Reader(bytes.NewReader(src))
			out, err := io.ReadAll(dr)
			require.NoError(t, err)
			require.NotContains(t, out, bad, "raw invalid byte must never leak as UTF-8")
			require.True(t, utf8.Valid(out), "output must be well-formed UTF-8")
			require.Equal(t, encWindows1252, dr.detectedEncoding(),
				"an invalid byte at the commit boundary must flip the whole buffer to Windows-1252")
			require.Equal(t, latin1ToUTF8(src), out,
				"the whole buffer must be reinterpreted as Windows-1252, not committed as UTF-8")
		})
	}
}

// smallPrefixThenDataErrReader emits a long run of ASCII ('a') bytes that is
// guaranteed to exceed the deferred reader's commit cap, then on a LATER read
// returns a final chunk of ASCII bytes together with a non-EOF error on the
// same Read (which io.Reader explicitly permits). It models a stream that
// commits to UTF-8 and only afterwards reports corruption alongside a last
// batch of bytes.
type smallPrefixThenDataErrReader struct {
	prefix int    // remaining ASCII prefix bytes to emit before the error chunk
	tail   []byte // final bytes delivered together with err
	err    error
	sent   bool
}

func (r *smallPrefixThenDataErrReader) Read(p []byte) (int, error) {
	if r.prefix > 0 {
		n := min(len(p), r.prefix)
		for i := range n {
			p[i] = 'a'
		}
		r.prefix -= n
		return n, nil
	}
	if !r.sent {
		r.sent = true
		n := copy(p, r.tail)
		return n, r.err
	}
	return 0, r.err
}

// TestDeferredLatin1ReaderCommittedSurfacesReadErrorWithoutTruncation pins the
// post-commit drain ordering. After the reader commits to UTF-8 (the prefix
// exceeds the cap), the underlying reader returns a final chunk of bytes
// together with a non-EOF error. Read with a buffer SMALLER than that chunk so
// the sanitizer buffers leftover converted output while the error is pending:
// every buffered byte must still drain before the error surfaces. The buggy
// ordering surfaced the saved error first and dropped the sanitizer's leftover.
func TestDeferredLatin1ReaderCommittedSurfacesReadErrorWithoutTruncation(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("checksum mismatch")

	const prefixLen = deferredLatin1MaxBuffer + 100 // forces a UTF-8 commit
	tail := bytes.Repeat([]byte("a"), 64)           // larger than the read buffer below

	src := &smallPrefixThenDataErrReader{prefix: prefixLen, tail: tail, err: sentinel}
	dr := newDeferredLatin1Reader(src)

	// A read buffer smaller than tail keeps converted output buffered inside the
	// sanitizer at the moment the sticky error is recorded.
	buf := make([]byte, 7)
	var out []byte
	var readErr error
	for {
		n, err := dr.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			readErr = err
			break
		}
	}

	require.ErrorIs(t, readErr, sentinel,
		"the post-commit non-EOF read error must surface")
	require.Len(t, out, prefixLen+len(tail),
		"every committed/sanitized byte must drain before the error truncates the stream")
	for _, b := range out {
		require.EqualValues(t, 'a', b)
	}
}

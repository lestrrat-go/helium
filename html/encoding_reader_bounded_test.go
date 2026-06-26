package html

import (
	"bytes"
	"io"
	"testing"

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
	dr := newDeferredLatin1Reader(src, "Windows-1252")

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
	dr := newDeferredLatin1Reader(src, "Windows-1252")

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

	dr := newDeferredLatin1Reader(bytes.NewReader(src.Bytes()), "Windows-1252")

	out, err := io.ReadAll(dr)
	require.NoError(t, err)
	require.NotContains(t, out, byte(0x93), "raw invalid byte must never leak into output")
	require.Contains(t, string(out), "�", "invalid byte must be replaced with U+FFFD")
	// Output = prefix + U+FFFD (3 bytes) + 'z'.
	require.Len(t, out, prefixSize+3+1, "all valid bytes delivered, invalid byte sanitized")
	require.Equal(t, byte('z'), out[len(out)-1], "bytes after the invalid one still flow through")
}

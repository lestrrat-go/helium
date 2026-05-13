package html

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// errShortInnerWriter is an io.Writer that always claims to write fewer
// bytes than requested while also reporting err == nil. A latin1EncodingWriter
// wrapping such an underlying writer must NOT silently swallow this: that
// would let serialized HTML be truncated without the caller knowing.
type errShortInnerWriter struct {
	written []byte
}

func (w *errShortInnerWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	// Accept only the first byte and pretend success.
	w.written = append(w.written, p[0])
	return 1, nil
}

func TestLatin1EncodingWriter_PromotesSilentShortWriteToError(t *testing.T) {
	inner := &errShortInnerWriter{}
	w := &latin1EncodingWriter{w: inner, strict: false}

	// All-ASCII input takes the 1:1 fast path: the inner accepted 1 byte
	// so exactly 1 byte of input was consumed. The wrapper must report
	// io.ErrShortWrite and the precise consumed count.
	input := []byte("hello world")
	n, err := w.Write(input)
	require.ErrorIs(t, err, io.ErrShortWrite,
		"wrapper must surface the silent short write")
	require.Equal(t, 1, n,
		"ASCII fast path is 1:1, so 1 inner byte == 1 input byte consumed")
}

// errInnerWriter returns a real error on every call.
type errInnerWriter struct{ err error }

func (w *errInnerWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

func TestLatin1EncodingWriter_PropagatesInnerError(t *testing.T) {
	myErr := errors.New("disk full")
	w := &latin1EncodingWriter{w: &errInnerWriter{err: myErr}, strict: false}

	n, err := w.Write([]byte("anything"))
	require.ErrorIs(t, err, myErr, "wrapper must propagate the inner writer's error")
	require.Equal(t, 0, n)
}

// boundedInnerWriter accepts up to maxBytes total across all Write calls and
// then short-writes by accepting only the remaining capacity. Used to drive
// the latin1 wrapper into a precise mid-conversion partial-write scenario.
type boundedInnerWriter struct {
	maxBytes int
	written  int
}

func (w *boundedInnerWriter) Write(p []byte) (int, error) {
	avail := w.maxBytes - w.written
	if avail <= 0 {
		return 0, nil
	}
	if len(p) <= avail {
		w.written += len(p)
		return len(p), nil
	}
	w.written = w.maxBytes
	return avail, nil
}

// When the inner writer commits a partial output, the wrapper must map back
// to the largest input rune boundary that fully fit within the inner's count
// — not return 0, which would mislead io.Copy-style callers into duplicating
// the bytes the inner already accepted.
func TestLatin1EncodingWriter_ShortWriteReportsRuneAlignedConsumed(t *testing.T) {
	// Input: "AéB" — three runes, UTF-8 lengths {1, 2, 1} = 4 input bytes.
	// Latin-1 output is {0x41, 0xE9, 0x42} = 3 output bytes (one byte per rune).
	// Inner cap = 2 output bytes → inner writes "A\xE9" successfully, then
	// short-writes on the trailing "B". The wrapper should report that the
	// first two runes were consumed: 1 + 2 = 3 input bytes.
	w := &latin1EncodingWriter{w: &boundedInnerWriter{maxBytes: 2}, strict: false}
	input := []byte("AéB")
	n, err := w.Write(input)
	require.ErrorIs(t, err, io.ErrShortWrite)
	require.Equal(t, 3, n,
		"two complete runes (1+2 input bytes) made it through the inner writer")
}

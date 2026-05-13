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

	// Input long enough that the inner writer's single-byte accept cannot
	// satisfy it. The wrapper must report io.ErrShortWrite and report 0
	// bytes of the original input consumed — never claim full success.
	input := []byte("hello world")
	n, err := w.Write(input)
	require.Error(t, err, "wrapper must surface the silent short write")
	require.True(t, errors.Is(err, io.ErrShortWrite),
		"expected io.ErrShortWrite, got %v", err)
	require.Equal(t, 0, n, "must not claim any input bytes consumed on error")
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

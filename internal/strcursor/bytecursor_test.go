package strcursor

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// dataThenErrReader returns its payload together with a non-EOF error on the
// same Read (which io.Reader explicitly permits), then reports EOF. It models a
// reader that detects corruption/truncation only after emitting the final
// bytes, e.g. a checksumming or decompressing stream.
type dataThenErrReader struct {
	data []byte
	err  error
	done bool
}

func (r *dataThenErrReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.data)
	return n, r.err
}

func TestByteCursorSurfacesErrorReturnedWithData(t *testing.T) {
	wantErr := errors.New("checksum mismatch")
	cur := NewByteCursor(&dataThenErrReader{
		data: []byte("<root/>"),
		err:  wantErr,
	})

	// The buffered bytes must remain readable.
	require.Equal(t, "<root/>", cur.PeekString(7))

	// Consume the buffered bytes.
	require.NoError(t, cur.Advance(7))

	// Once the buffer drains, the cursor must report the underlying error
	// rather than silently treating the stream as cleanly terminated.
	require.True(t, cur.Done(), "Done should be true after buffer drains")
	require.ErrorIs(t, cur.Err(), wantErr, "the non-EOF read error must be surfaced after the buffered bytes are consumed")
}

// zeroProgressReader always returns (0, nil) for a non-empty request, never
// advancing and never erroring. A naive fill loop spins on it forever; the
// bounded-retry fill loop must give up after maxZeroProgressReads.
type zeroProgressReader struct{}

func (zeroProgressReader) Read(p []byte) (int, error) {
	return 0, nil
}

func TestByteCursorZeroProgressReaderDoesNotHang(t *testing.T) {
	cur := NewByteCursor(zeroProgressReader{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		require.True(t, cur.Done(), "a zero-progress reader must terminate fill, not spin")
		require.ErrorIs(t, cur.Err(), io.ErrNoProgress, "a zero-progress reader must surface io.ErrNoProgress after the bounded retry count")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ByteCursor fillBuffer hung on a zero-progress reader")
	}
}

// slowSplitReader emits its payload one byte per Read, returning a single
// (0, nil) "waiting" read before each real byte. A fill loop that treats the
// first (0, nil) as fatal would reject this legitimate slow producer.
type slowSplitReader struct {
	data    []byte
	pos     int
	pending bool // true after a (0, nil) stall, so the next Read yields a byte
}

func (r *slowSplitReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	if !r.pending {
		r.pending = true
		return 0, nil // stall once before delivering the next byte
	}
	r.pending = false
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

func TestByteCursorSlowSplitReaderMakesProgress(t *testing.T) {
	cur := NewByteCursor(&slowSplitReader{data: []byte("<root/>")})

	done := make(chan struct{})
	go func() {
		defer close(done)
		require.True(t, cur.HasPrefix([]byte("<root/>")), "a slow reader that emits (0, nil) between bytes must still be consumed")
		require.NoError(t, cur.Err(), "a progressing reader must not surface io.ErrNoProgress")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ByteCursor fillBuffer hung on a slow split reader")
	}
}

func TestByteCursorTreatsEOFWithDataAsCleanEnd(t *testing.T) {
	cur := NewByteCursor(&dataThenErrReader{
		data: []byte("<root/>"),
		err:  io.EOF,
	})

	require.Equal(t, "<root/>", cur.PeekString(7))
	require.NoError(t, cur.Advance(7))
	require.True(t, cur.Done())
	require.NoError(t, cur.Err(), "io.EOF must not be reported as an error")
}

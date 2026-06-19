package push

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStreamReadReturnsAvailableBytes verifies that a single small Write
// makes Read return those bytes promptly, without waiting for a full
// len(p) chunk or for the stream to be closed. This is the core of
// incremental push parsing.
func TestStreamReadReturnsAvailableBytes(t *testing.T) {
	t.Parallel()

	s := newStream(t.Context())

	const payload = "hi"
	_, err := s.Write([]byte(payload))
	require.NoError(t, err)

	type readResult struct {
		n   int
		err error
		buf []byte
	}
	resCh := make(chan readResult, 1)
	go func() {
		// len(p) is much larger than the written payload; Read must not
		// block waiting for the buffer to fill.
		p := make([]byte, 4096)
		n, err := s.Read(p)
		resCh <- readResult{n: n, err: err, buf: p[:n]}
	}()

	select {
	case res := <-resCh:
		require.NoError(t, res.err)
		require.Equal(t, len(payload), res.n)
		require.Equal(t, payload, string(res.buf))
	case <-time.After(2 * time.Second):
		t.Fatal("Read blocked instead of returning available bytes")
	}
}

// TestStreamReadBlocksWhileEmpty verifies that Read blocks while the
// buffer is empty and the stream is open, then unblocks once data is
// written.
func TestStreamReadBlocksWhileEmpty(t *testing.T) {
	t.Parallel()

	s := newStream(t.Context())

	resCh := make(chan int, 1)
	go func() {
		p := make([]byte, 16)
		n, err := s.Read(p)
		require.NoError(t, err)
		resCh <- n
	}()

	// Read should still be blocked since nothing was written.
	select {
	case <-resCh:
		t.Fatal("Read returned before any data was available")
	case <-time.After(100 * time.Millisecond):
	}

	_, err := s.Write([]byte("data"))
	require.NoError(t, err)

	select {
	case n := <-resCh:
		require.Equal(t, 4, n)
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not unblock after Write")
	}
}

// TestStreamReadContextCancellation verifies that cancelling the context
// aborts a Read that is blocked on an empty, open stream.
func TestStreamReadContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	s := newStream(ctx)

	errCh := make(chan error, 1)
	go func() {
		p := make([]byte, 16)
		_, err := s.Read(p)
		errCh <- err
	}()

	// Ensure the reader is parked on cond.Wait before cancelling.
	select {
	case <-errCh:
		t.Fatal("Read returned before cancellation")
	case <-time.After(100 * time.Millisecond):
	}

	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not abort after context cancellation")
	}
}

// TestStreamReadEOFAfterClose verifies that a Read on an empty, closed
// stream returns io.EOF.
func TestStreamReadEOFAfterClose(t *testing.T) {
	t.Parallel()

	s := newStream(t.Context())
	require.NoError(t, s.close())

	p := make([]byte, 16)
	n, err := s.Read(p)
	require.Equal(t, 0, n)
	require.ErrorIs(t, err, io.EOF)
}

// TestStreamReadDrainsBeforeEOF verifies that buffered data is fully
// drained before EOF is reported, even after the stream is closed.
func TestStreamReadDrainsBeforeEOF(t *testing.T) {
	t.Parallel()

	s := newStream(t.Context())
	_, err := s.Write([]byte("tail"))
	require.NoError(t, err)
	require.NoError(t, s.close())

	p := make([]byte, 16)
	n, err := s.Read(p)
	require.NoError(t, err)
	require.Equal(t, "tail", string(p[:n]))

	n, err = s.Read(p)
	require.Equal(t, 0, n)
	require.ErrorIs(t, err, io.EOF)
}

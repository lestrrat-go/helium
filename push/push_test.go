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

	type readResult struct {
		n   int
		err error
	}
	resCh := make(chan readResult, 1)
	go func() {
		p := make([]byte, 16)
		n, err := s.Read(p)
		resCh <- readResult{n: n, err: err}
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
	case res := <-resCh:
		require.NoError(t, res.err)
		require.Equal(t, 4, res.n)
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

// TestStreamReadZeroLength verifies that a zero-length Read on an open,
// empty stream returns (0, nil) promptly instead of blocking, as required
// by the io.Reader contract.
func TestStreamReadZeroLength(t *testing.T) {
	t.Parallel()

	s := newStream(t.Context())

	type readResult struct {
		n   int
		err error
	}
	resCh := make(chan readResult, 1)
	go func() {
		n, err := s.Read(nil)
		resCh <- readResult{n: n, err: err}
	}()

	select {
	case res := <-resCh:
		require.NoError(t, res.err)
		require.Equal(t, 0, res.n)
	case <-time.After(2 * time.Second):
		t.Fatal("zero-length Read blocked instead of returning promptly")
	}
}

// TestNewNilContext verifies that New tolerates a nil context (as passed
// by NewPushParser(nil)) without panicking.
func TestNewNilContext(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		//nolint:staticcheck // intentionally exercising the nil-context path
		pp := New[struct{}](nil, nopSource{})
		_, _ = pp.Close()
	})
}

// nopSource is a trivial Source that consumes the stream and returns a
// zero value, used to exercise New without a real parser.
type nopSource struct{}

func (nopSource) ParseReader(_ context.Context, r io.Reader) (struct{}, error) {
	_, _ = io.Copy(io.Discard, r)
	return struct{}{}, nil
}

// finishSignalSource models a parser that completes successfully without ever
// observing context cancellation through the stream. It performs a single Read,
// signals on parsed, then blocks on resume so the test can cancel the context
// and have that cancel be visible to the parse epilogue BEFORE it runs. A
// correct epilogue must still report success because no Read returned the
// context error.
type finishSignalSource struct {
	result string
	parsed chan struct{}
	resume chan struct{}
}

func (s finishSignalSource) ParseReader(_ context.Context, r io.Reader) (string, error) {
	p := make([]byte, 4096)
	_, _ = r.Read(p)
	close(s.parsed)
	<-s.resume
	return s.result, nil
}

// TestNewLateCancelKeepsResult verifies that a parse which completes
// successfully is still reported as success when the context is cancelled
// immediately AFTER ParseReader returns. The late cancel must not be turned
// into context.Canceled by the parse epilogue, since no Read ever observed it.
func TestNewLateCancelKeepsResult(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	const want = "parsed-document"
	src := finishSignalSource{
		result: want,
		parsed: make(chan struct{}),
		resume: make(chan struct{}),
	}

	pp := New[string](ctx, src)
	require.NoError(t, pp.Push([]byte("payload")))

	// Wait until the parse has done its work, then cancel the context so the
	// cancel is observable to the epilogue before it runs, then let the parse
	// return. This deterministically reproduces a late external cancel.
	select {
	case <-src.parsed:
	case <-time.After(2 * time.Second):
		t.Fatal("ParseReader did not start")
	}
	cancel()
	close(src.resume)

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// TestStreamReadEOFAfterClose verifies that a Read on an empty, closed
// stream returns io.EOF.
func TestStreamReadEOFAfterClose(t *testing.T) {
	t.Parallel()

	s := newStream(t.Context())
	s.close()

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
	s.close()

	p := make([]byte, 16)
	n, err := s.Read(p)
	require.NoError(t, err)
	require.Equal(t, "tail", string(p[:n]))

	n, err = s.Read(p)
	require.Equal(t, 0, n)
	require.ErrorIs(t, err, io.EOF)
}

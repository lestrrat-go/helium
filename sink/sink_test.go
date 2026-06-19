package sink_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/sink"
	"github.com/stretchr/testify/require"
)

// compile-time assertions
var _ io.Closer = (*sink.Sink[error])(nil)

func TestSinkHandleAndClose(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	var mu sync.Mutex
	var got []string

	s := sink.New[string](ctx, sink.HandlerFunc[string](func(_ context.Context, v string) {
		mu.Lock()
		got = append(got, v)
		mu.Unlock()
	}))

	want := []string{"a", "b", "c"}
	for _, v := range want {
		s.Handle(ctx, v)
	}

	require.NoError(t, s.Close())

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, want, got)
}

func TestSinkCloseDrains(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	var mu sync.Mutex
	var got []int

	s := sink.New[int](ctx, sink.HandlerFunc[int](func(_ context.Context, v int) {
		mu.Lock()
		got = append(got, v)
		mu.Unlock()
	}), sink.WithBufferSize(16))

	for i := range 10 {
		s.Handle(ctx, i)
	}

	require.NoError(t, s.Close())

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, got, 10)
}

func TestSinkCloseMultipleTimes(t *testing.T) {
	t.Parallel()

	s := sink.New[int](t.Context(), sink.HandlerFunc[int](func(_ context.Context, _ int) {}))
	require.NoError(t, s.Close())
	require.NoError(t, s.Close())
}

func TestSinkNegativeBufferSizeDoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	var mu sync.Mutex
	var got []int

	var s *sink.Sink[int]
	require.NotPanics(t, func() {
		s = sink.New[int](ctx, sink.HandlerFunc[int](func(_ context.Context, v int) {
			mu.Lock()
			got = append(got, v)
			mu.Unlock()
		}), sink.WithBufferSize(-1))
	})

	s.Handle(ctx, 1)
	s.Handle(ctx, 2)
	require.NoError(t, s.Close())

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []int{1, 2}, got)
}

func TestSinkCloseWhileHandleBlockedUnblocksHandle(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})

	s := sink.New[string](ctx, sink.HandlerFunc[string](func(_ context.Context, v string) {
		if v != "first" {
			return
		}
		close(handlerStarted)
		<-releaseHandler
	}), sink.WithBufferSize(1))

	s.Handle(ctx, "first")
	<-handlerStarted
	s.Handle(ctx, "second")

	panicCh := make(chan any, 1)
	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		defer func() {
			panicCh <- recover()
		}()
		s.Handle(ctx, "third")
	}()

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- s.Close()
	}()

	<-handleDone
	close(releaseHandler)

	require.NoError(t, <-closeDone)
	require.Nil(t, <-panicCh)
}

func TestSinkNilReceiver(t *testing.T) {
	t.Parallel()

	var s *sink.Sink[error]
	s.Handle(t.Context(), errors.New("test"))
	require.NoError(t, s.Close())
}

func TestSinkCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	var mu sync.Mutex
	var got []string

	s := sink.New[string](ctx, sink.HandlerFunc[string](func(_ context.Context, v string) {
		mu.Lock()
		got = append(got, v)
		mu.Unlock()
	}))

	s.Handle(ctx, "before-cancel")
	cancel()

	require.NoError(t, s.Close())

	mu.Lock()
	defer mu.Unlock()
	require.Contains(t, got, "before-cancel")
}

func TestSinkNilHandlerDoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var s *sink.Sink[string]
	require.NotPanics(t, func() {
		s = sink.New[string](ctx, nil)
	})

	// Delivering an item must not panic even though no real handler was given.
	require.NotPanics(t, func() {
		s.Handle(ctx, "first")
		s.Handle(ctx, "second")
	})

	require.NoError(t, s.Close())
}

func TestSinkNilHandlerDeliveryDrainsWithoutPanic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	s := sink.New[int](ctx, nil, sink.WithBufferSize(4))
	for i := range 8 {
		s.Handle(ctx, i)
	}

	done := make(chan error, 1)
	go func() { done <- s.Close() }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Close on a nil-handler sink deadlocked")
	}
}

func TestSinkSelfCloseDoesNotDeadlock(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var s *sink.Sink[string]
	closeErr := make(chan error, 1)

	s = sink.New[string](ctx, sink.HandlerFunc[string](func(_ context.Context, _ string) {
		// A handler that closes its own sink from within Handle must not
		// deadlock: Close is expected to return promptly.
		closeErr <- s.Close()
	}))

	s.Handle(ctx, "trigger")

	select {
	case err := <-closeErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("self-close from within Handler deadlocked")
	}

	// A subsequent external Close must also return promptly and not deadlock.
	extClose := make(chan error, 1)
	go func() { extClose <- s.Close() }()
	select {
	case err := <-extClose:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("external Close after self-close deadlocked")
	}
}

func TestSinkReentrantHandleDoesNotDeadlock(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var s *sink.Sink[int]
	handled := make(chan int, 16)

	s = sink.New[int](ctx, sink.HandlerFunc[int](func(c context.Context, v int) {
		handled <- v
		// Re-emit a derived item from within Handle. This must not block the
		// worker goroutine on its own (possibly full) buffer.
		if v < 3 {
			s.Handle(c, v+1)
		}
	}), sink.WithBufferSize(1))

	s.Handle(ctx, 0)

	closeDone := make(chan error, 1)
	go func() { closeDone <- s.Close() }()

	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("re-entrant Handle from within Handler deadlocked")
	}

	require.NotEmpty(t, handled)
}

func TestSinkErrorSatisfiesErrorHandler(t *testing.T) {
	t.Parallel()

	// ErrorHandler is Handle(context.Context, error)
	// *sink.Sink[error] has Handle(context.Context, error)
	ctx := t.Context()
	var mu sync.Mutex
	var got []error

	s := sink.New[error](ctx, sink.HandlerFunc[error](func(_ context.Context, err error) {
		mu.Lock()
		got = append(got, err)
		mu.Unlock()
	}))

	testErr := errors.New("test error")
	s.Handle(ctx, testErr)
	require.NoError(t, s.Close())

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []error{testErr}, got)
}

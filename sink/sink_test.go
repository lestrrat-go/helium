package sink_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

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

func TestSinkCloseWhileHandleBlockedDoesNotPanic(t *testing.T) {
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

	handleCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	panicCh := make(chan any, 1)
	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		defer func() {
			panicCh <- recover()
		}()
		s.Handle(handleCtx, "third")
	}()

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- s.Close()
	}()

	cancel()
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

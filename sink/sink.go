package sink

import (
	"context"
	"sync"
)

// Handler processes items delivered to a Sink's background goroutine.
// When T is error, this interface has the same signature as helium.ErrorHandler.
type Handler[T any] interface {
	Handle(context.Context, T)
}

// HandlerFunc is an adapter to use a plain function as a Handler.
type HandlerFunc[T any] func(context.Context, T)

func (f HandlerFunc[T]) Handle(ctx context.Context, data T) {
	f(ctx, data)
}

// Option configures a Sink.
type Option func(*config)

type config struct {
	bufSize int
}

// WithBufferSize sets the channel buffer size. Default is 256.
func WithBufferSize(n int) Option {
	return func(c *config) { c.bufSize = n }
}

// Sink is a generic, channel-based event sink. Items are sent via Handle()
// and processed asynchronously by a Handler in a background goroutine.
//
// A nil *Sink is safe to use — Handle() is a no-op on a nil receiver.
//
// When T is error, *Sink[error] satisfies the helium.ErrorHandler interface.
type Sink[T any] struct {
	ch      chan T
	handler Handler[T]
	done    chan struct{}
	once    sync.Once
}

// New creates a Sink that delivers items to handler in a background
// goroutine. The goroutine exits when Close() is called or ctx is cancelled.
// In both cases, buffered items are drained before the goroutine exits.
func New[T any](ctx context.Context, handler Handler[T], options ...Option) *Sink[T] {
	cfg := config{bufSize: 256}
	for _, o := range options {
		o(&cfg)
	}
	s := &Sink[T]{
		ch:      make(chan T, cfg.bufSize),
		handler: handler,
		done:    make(chan struct{}),
	}
	go s.start(ctx)
	return s
}

// Handle sends data to the sink for async processing. If the buffer is full,
// Handle blocks until space is available or ctx is cancelled.
// Safe to call on a nil receiver (no-op).
func (s *Sink[T]) Handle(ctx context.Context, data T) {
	if s == nil {
		return
	}
	select {
	case s.ch <- data:
	case <-ctx.Done():
	}
}

// Close stops the sink and waits for all buffered items to be processed.
// Safe to call on a nil receiver (no-op). Safe to call multiple times.
func (s *Sink[T]) Close() error {
	if s == nil {
		return nil
	}
	s.once.Do(func() { close(s.ch) })
	<-s.done
	return nil
}

func (s *Sink[T]) start(ctx context.Context) {
	defer close(s.done)
	for {
		select {
		case data, ok := <-s.ch:
			if !ok {
				return // channel closed via Close()
			}
			s.handler.Handle(ctx, data)
		case <-ctx.Done():
			// context cancelled — drain remaining buffered items
			s.once.Do(func() { close(s.ch) })
			for data := range s.ch {
				s.handler.Handle(ctx, data)
			}
			return
		}
	}
}

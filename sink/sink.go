package sink

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
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

// WithBufferSize sets the channel buffer size. Default is 256. Negative
// values are clamped to 0 (an unbuffered channel).
func WithBufferSize(n int) Option {
	return func(c *config) { c.bufSize = n }
}

// Sink is a generic, channel-based event sink. Items are sent via Handle()
// and processed asynchronously by a Handler in a background goroutine.
//
// A nil *Sink is safe to use — Handle() is a no-op on a nil receiver. A Sink
// created with a nil Handler is also safe — items are discarded rather than
// delivered, so delivery never panics.
//
// When T is error, *Sink[error] satisfies the helium.ErrorHandler interface.
type Sink[T any] struct {
	ch      chan T
	handler Handler[T]
	done    chan struct{}
	closing chan struct{}
	once    sync.Once
	mu      sync.RWMutex
	closed  bool
	senders sync.WaitGroup
	// workerID holds the goroutine id of the background worker. It is set
	// once when the worker starts and read by Close to detect a re-entrant
	// Close issued from within a Handler (which would otherwise deadlock on
	// done). A value of 0 means "not yet set".
	workerID atomic.Uint64
}

// New creates a Sink that delivers items to handler in a background
// goroutine. The goroutine exits when Close() is called or ctx is cancelled.
// In both cases, buffered items are drained before the goroutine exits.
//
// A nil handler is replaced with a no-op handler that discards items, so the
// returned Sink is always safe to deliver to and never panics on delivery.
func New[T any](ctx context.Context, handler Handler[T], options ...Option) *Sink[T] {
	if handler == nil {
		handler = HandlerFunc[T](func(context.Context, T) {})
	}
	cfg := config{bufSize: 256}
	for _, o := range options {
		o(&cfg)
	}
	if cfg.bufSize < 0 {
		cfg.bufSize = 0
	}
	s := &Sink[T]{
		ch:      make(chan T, cfg.bufSize),
		handler: handler,
		done:    make(chan struct{}),
		closing: make(chan struct{}),
	}
	go s.start(ctx)
	return s
}

// Handle sends data to the sink for async processing. If the buffer is full,
// Handle blocks until space is available, ctx is cancelled, or the sink closes.
// Safe to call on a nil receiver (no-op).
//
// Handle may be called re-entrantly from within a Handler. In that case it does
// a non-blocking best-effort send (dropping the item if the buffer is full or
// the sink is closing) rather than blocking: the calling goroutine is the
// worker itself, so blocking on its own buffer would deadlock and would also
// stall a concurrent Close that waits on in-flight senders.
func (s *Sink[T]) Handle(ctx context.Context, data T) {
	if s == nil {
		return
	}
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return
	}
	// Register as an in-flight sender before releasing the lock. This blocks
	// shutdown from closing ch until the send below completes, which both
	// prevents a send-on-closed-channel panic and avoids a data race between
	// the send and close(ch).
	s.senders.Add(1)
	ch := s.ch
	closing := s.closing
	s.mu.RUnlock()
	defer s.senders.Done()

	// A re-entrant send from the worker goroutine must never block: the worker
	// is the only goroutine that drains ch, so blocking on a full buffer would
	// deadlock it (and stall any concurrent Close waiting on this sender). Fall
	// back to a non-blocking best-effort send, dropping the item if the buffer
	// is full or the sink is closing.
	if id := s.workerID.Load(); id != 0 && id == goroutineID() {
		select {
		case <-closing:
		case ch <- data:
		default:
		}
		return
	}

	select {
	case <-closing:
	case ch <- data:
	case <-ctx.Done():
	}
}

// Close stops the sink and waits for all buffered items to be processed.
// Safe to call on a nil receiver (no-op). Safe to call multiple times.
//
// Close is safe to call from within a Handler (a "self-close"). In that case
// it initiates shutdown and returns immediately instead of waiting on the
// worker goroutine — waiting would deadlock, because the worker is the very
// goroutine executing the Handler that called Close. The worker drains any
// remaining buffered items and exits once the active Handler returns.
func (s *Sink[T]) Close() error {
	if s == nil {
		return nil
	}
	s.shutdown()
	// Detect a re-entrant Close issued from the worker goroutine itself.
	// Blocking on done in that case is a guaranteed deadlock, so skip the
	// wait and let the worker unwind naturally.
	if id := s.workerID.Load(); id != 0 && id == goroutineID() {
		return nil
	}
	<-s.done
	return nil
}

func (s *Sink[T]) shutdown() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.closing)
		s.mu.Unlock()
		s.senders.Wait()
		close(s.ch)
	})
}

func (s *Sink[T]) start(ctx context.Context) {
	s.workerID.Store(goroutineID())
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
			s.shutdown()
			for data := range s.ch {
				s.handler.Handle(ctx, data)
			}
			return
		}
	}
}

// goroutineID returns the id of the calling goroutine. It is used solely to
// detect a re-entrant Close issued from the worker goroutine, where blocking
// on done would deadlock. It is best-effort: if the runtime stack header
// cannot be parsed, it returns 0, which simply disables the optimization and
// preserves the (correct, blocking) behavior for all other callers.
func goroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// The stack header looks like: "goroutine 123 [running]:\n..."
	const prefix = "goroutine "
	b := buf[:n]
	if len(b) < len(prefix) {
		return 0
	}
	b = b[len(prefix):]
	var id uint64
	var saw bool
	for _, c := range b {
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + uint64(c-'0')
		saw = true
	}
	if !saw {
		return 0
	}
	return id
}

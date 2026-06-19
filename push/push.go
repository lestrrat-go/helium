// Package push provides a generic push parser that accepts data in
// chunks and parses it in a background goroutine. It is used by both the
// XML ([helium.Parser]) and HTML ([html.Parser]) push-parser APIs.
//
// The [ReaderParser] interface abstracts the underlying parser; any type
// with a ParseReader(ctx, io.Reader) method can be used.
package push

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// stream is a thread-safe concurrent buffer. Write appends data and
// signals waiting readers; Read blocks only while the buffer is empty
// and the stream is neither closed nor its context cancelled, then
// returns whatever bytes are currently available (up to len(p)). This
// enables incremental push parsing: a reader is woken as soon as any
// data arrives instead of stalling until a full len(p) chunk or EOF.
type stream struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    bytes.Buffer
	closed bool
	wrErr  error
	// ctxErr reports the parse context's cancellation error, if any. It is
	// a closure rather than a stored context.Context to keep the runtime
	// context out of long-lived struct state.
	ctxErr  func() error
	stopped chan struct{}
}

func newStream(ctx context.Context) *stream {
	s := &stream{ctxErr: ctx.Err, stopped: make(chan struct{})}
	s.cond = sync.NewCond(&s.mu)
	// Watch the context so a blocked Read is woken on cancellation. The
	// Broadcast happens under the lock so it can never be lost between a
	// reader's wait-condition check and its cond.Wait(). The watcher also
	// exits once the stream is closed, so it does not leak when ctx is
	// never cancelled.
	go func() {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			s.cond.Broadcast()
			s.mu.Unlock()
		case <-s.stopped:
		}
	}()
	return s
}

func (s *stream) Read(p []byte) (int, error) {
	// Per the io.Reader contract a zero-length Read must return (0, nil)
	// immediately rather than block on the wait loop below.
	if len(p) == 0 {
		return 0, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for s.buf.Len() == 0 && !s.closed && s.ctxErr() == nil {
		s.cond.Wait()
	}

	if err := s.ctxErr(); err != nil {
		return 0, err
	}

	if s.buf.Len() == 0 && s.closed {
		return 0, io.EOF
	}

	return s.buf.Read(p)
}

func (s *stream) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.wrErr != nil {
		return 0, s.wrErr
	}
	if s.closed {
		return 0, io.ErrClosedPipe
	}

	n, err := s.buf.Write(p)
	s.cond.Signal()
	return n, err
}

func (s *stream) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.markClosed()
	s.cond.Broadcast()
	return nil
}

func (s *stream) closeWithWriteError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.wrErr = err
	s.markClosed()
	s.cond.Broadcast()
}

// markClosed flips the closed flag and stops the context watcher exactly
// once. Caller must hold s.mu.
func (s *stream) markClosed() {
	if s.closed {
		return
	}
	s.closed = true
	close(s.stopped)
}

// Source is the interface satisfied by any parser that can parse from
// an io.Reader and return a result of type T.
type Source[T any] interface {
	ParseReader(ctx context.Context, r io.Reader) (T, error)
}

// Parser manages a background goroutine that parses data pushed via
// [Parser.Push] or [Parser.Write]. T is the result type produced by
// the underlying parser. Call [Parser.Close] to signal end-of-input
// and retrieve the result.
type Parser[T any] struct {
	s         *stream
	done      chan result[T]
	closeOnce sync.Once
	res       result[T]
}

type result[T any] struct {
	val T
	err error
}

// New creates a Parser and starts a background goroutine that feeds
// pushed data to the given [ReaderParser]. The goroutine recovers from
// panics and delivers the result when [Parser.Close] is called.
func New[T any](ctx context.Context, p Source[T]) *Parser[T] {
	// Normalize a nil context so callers such as NewPushParser(nil) do not
	// panic when newStream dereferences ctx.Err/ctx.Done. There is no parent
	// to inherit from here (the parent context is nil), so context.Background
	// is the correct root; contextcheck's "non-inherited new context" warning
	// does not apply.
	if ctx == nil {
		ctx = context.Background() //nolint:contextcheck
	}
	s := newStream(ctx)
	pp := &Parser[T]{
		s:    s,
		done: make(chan result[T], 1),
	}
	go func() {
		var res result[T]
		defer func() {
			if r := recover(); r != nil {
				if res.err == nil {
					res.err = fmt.Errorf("parser panic: %v", r)
				}
			}
			// A cancelled context must surface through Close even if the
			// underlying parser swallowed the stream's ctx error (some
			// readers treat any non-nil read error as EOF and return a
			// partial, error-free result).
			if res.err == nil && ctx.Err() != nil {
				var zero T
				res = result[T]{val: zero, err: ctx.Err()}
			}
			s.closeWithWriteError(res.err)
			pp.done <- res
		}()
		res.val, res.err = p.ParseReader(ctx, s)
	}()
	return pp
}

// Push sends a chunk of data to the parser. It returns an error if the
// parser has already failed or if the stream has been closed.
func (pp *Parser[T]) Push(chunk []byte) error {
	_, err := pp.s.Write(chunk)
	return err
}

// Write implements io.Writer, allowing use with io.Copy and similar functions.
func (pp *Parser[T]) Write(p []byte) (int, error) {
	return pp.s.Write(p)
}

// Close signals end-of-input, waits for the parse goroutine to finish,
// and returns the parsed result. It is idempotent: subsequent calls
// return the same result.
func (pp *Parser[T]) Close() (T, error) {
	pp.closeOnce.Do(func() {
		if err := pp.s.close(); err != nil {
			pp.res = result[T]{err: err}
			return
		}
		pp.res = <-pp.done
	})
	return pp.res.val, pp.res.err
}

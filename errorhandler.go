package helium

import (
	"context"
	"sync"

	"github.com/lestrrat-go/helium/sink"
)

// ErrorLeveler is an optional interface that errors can implement to
// report their severity. Errors that do not implement this interface
// are treated as warnings (ErrorLevelWarning).
type ErrorLeveler interface {
	ErrorLevel() ErrorLevel
}

// ErrorHandler receives errors reported during parsing, compilation,
// or validation. Implementations may log, accumulate, or discard errors.
//
// Handle is called synchronously at the point of error detection unless
// the implementation itself introduces asynchrony (e.g. Sink[error]).
//
// Implementations must not block for extended periods.
//
// The error value may optionally implement ErrorLeveler to indicate
// severity. Users can type-assert to inspect the level.
type ErrorHandler interface {
	Handle(context.Context, error)
}

// NilErrorHandler is an ErrorHandler that discards all errors.
// Use as a default when no handler is provided.
type NilErrorHandler struct{}

func (NilErrorHandler) Handle(context.Context, error) {}

type errorAccumulator struct {
	level  ErrorLevel
	mu     sync.Mutex
	errors []error
}

func (a *errorAccumulator) Handle(_ context.Context, err error) {
	if a.level != 0 {
		level := ErrorLevelWarning
		if l, ok := err.(ErrorLeveler); ok {
			level = l.ErrorLevel()
		}
		if level != a.level {
			return
		}
	}
	a.mu.Lock()
	a.errors = append(a.errors, err)
	a.mu.Unlock()
}

func (a *errorAccumulator) collectErrors() []error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]error(nil), a.errors...)
}

// ErrorCollector collects errors into a slice via an internal Sink[error].
// When level is zero (ErrorLevelNone), all errors are collected. When set,
// only errors matching that level are collected.
//
// Satisfies ErrorHandler and io.Closer. The parser/compiler closes it
// automatically at the end of the operation.
type ErrorCollector struct {
	acc *errorAccumulator
	s   *sink.Sink[error]
}

// NewErrorCollector creates an ErrorCollector backed by a Sink[error].
// Pass ErrorLevelNone (0) for level to collect all errors regardless of severity.
func NewErrorCollector(ctx context.Context, level ErrorLevel, opts ...sink.Option) *ErrorCollector {
	acc := &errorAccumulator{level: level}
	return &ErrorCollector{
		acc: acc,
		s:   sink.New[error](ctx, acc, opts...),
	}
}

// Handle satisfies ErrorHandler. Sends the error to the internal Sink.
func (ec *ErrorCollector) Handle(ctx context.Context, err error) {
	ec.s.Handle(ctx, err)
}

// Close satisfies io.Closer. Drains the internal Sink.
func (ec *ErrorCollector) Close() error {
	return ec.s.Close()
}

// Errors returns a copy of the collected errors. Safe to call after Close.
func (ec *ErrorCollector) Errors() []error {
	return ec.acc.collectErrors()
}

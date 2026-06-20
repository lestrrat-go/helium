// Package sink provides a generic, channel-based asynchronous event sink.
//
// A Sink delivers items of type T to a [Handler] on a background goroutine,
// decoupling producers from consumers:
//
//	s := sink.New[error](ctx, sink.HandlerFunc[error](func(ctx context.Context, err error) {
//	    log.Println(err)
//	}))
//	s.Handle(ctx, fmt.Errorf("something happened"))
//	s.Close()
//
// When T is error, [*Sink] satisfies the [helium.ErrorHandler] interface,
// making it usable as an async error collector during parsing and validation.
//
// The buffer size defaults to 256 and can be configured via [WithBufferSize].
// Handle on a nil *Sink is a no-op, and a Sink built with a nil Handler
// discards items rather than panicking. A Handler may safely call Close or
// Handle on its own Sink; neither deadlocks.
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with sink_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package sink

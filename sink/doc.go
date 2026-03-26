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
// Handle on a nil *Sink is a no-op.
package sink

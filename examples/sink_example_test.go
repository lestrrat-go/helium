package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/sink"
)

func Example_sink_new() {
	// Sink[T] is a generic, channel-based async processor. Items sent via
	// Handle are delivered to a Handler in a background goroutine.
	//
	// When T is error, *Sink[error] satisfies the helium.ErrorHandler
	// interface via structural typing.

	ctx := context.Background()

	// Create a sink that collects strings.
	var collected []string

	// HandlerFunc adapts a plain function into a sink.Handler.
	handler := sink.HandlerFunc[string](func(_ context.Context, v string) {
		collected = append(collected, v)
	})

	// sink.New starts a background goroutine that reads from an internal
	// channel and calls the handler for each item.
	s := sink.New[string](ctx, handler)

	for _, v := range []string{"alpha", "bravo", "charlie"} {
		s.Handle(ctx, v)
	}

	// Close waits for all buffered items to be processed.
	_ = s.Close()

	for _, v := range collected {
		fmt.Println(v)
	}
	// Output:
	// alpha
	// bravo
	// charlie
}

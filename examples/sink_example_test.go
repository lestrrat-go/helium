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

	// Create a sink that collects strings. HandlerFunc adapts a plain
	// function into a Handler.
	var collected []string
	s := sink.New[string](ctx, sink.HandlerFunc[string](func(_ context.Context, v string) {
		collected = append(collected, v)
	}))

	s.Handle(ctx, "alpha")
	s.Handle(ctx, "bravo")
	s.Handle(ctx, "charlie")

	// Close waits for all buffered items to be processed.
	s.Close()

	for _, v := range collected {
		fmt.Println(v)
	}
	// Output:
	// alpha
	// bravo
	// charlie
}

package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_error_collector() {
	// ErrorCollector accumulates errors reported via its Handle method.
	// It wraps a Sink[error] internally and satisfies the ErrorHandler
	// and io.Closer interfaces.

	ctx := context.Background()

	// Collect all errors regardless of severity (pass ErrorLevelNone).
	collector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

	// Report some errors. In real usage the parser/compiler calls Handle.
	for _, msg := range []string{"first warning", "second warning"} {
		collector.Handle(ctx, errors.New(msg))
	}

	// Close drains any buffered errors from the internal sink.
	_ = collector.Close()

	// Errors() returns all collected errors.
	for _, e := range collector.Errors() {
		fmt.Println(e)
	}
	// Output:
	// first warning
	// second warning
}

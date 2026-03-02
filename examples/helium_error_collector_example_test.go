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
	collector.Handle(ctx, errors.New("first warning"))
	collector.Handle(ctx, errors.New("second warning"))

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

func Example_helium_error_collector_filter() {
	// ErrorCollector can filter errors by severity level. Here we
	// collect only warnings, ignoring fatal errors.
	//
	// Errors that implement ErrorLeveler report their own severity.
	// Errors that don't are treated as warnings by default.

	ctx := context.Background()

	// Collect warnings only.
	warnings := helium.NewErrorCollector(ctx, helium.ErrorLevelWarning)

	// A plain error (no ErrorLeveler) defaults to warning level.
	warnings.Handle(ctx, errors.New("a plain warning"))

	// An ErrParseError carries its level explicitly.
	warnings.Handle(ctx, helium.ErrParseError{
		Err:   errors.New("fatal parse error"),
		Level: helium.ErrorLevelFatal,
	})

	_ = warnings.Close()

	// Only the warning was collected; the fatal error was filtered out.
	fmt.Printf("collected: %d\n", len(warnings.Errors()))
	fmt.Println(warnings.Errors()[0])
	// Output:
	// collected: 1
	// a plain warning
}

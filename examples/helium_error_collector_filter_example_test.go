package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
)

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

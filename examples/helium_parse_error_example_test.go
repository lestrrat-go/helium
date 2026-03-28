package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse_error_inspect() {
	// When the parser encounters malformed XML, it returns an
	// ErrParseError containing the source location and context.
	// Use errors.AsType to extract the structured error.

	_, err := helium.NewParser().Parse(context.Background(), []byte(`<root>`))

	if pe, ok := errors.AsType[helium.ErrParseError](err); ok {
		fmt.Printf("line: %d\n", pe.LineNumber)
		fmt.Printf("column: %d\n", pe.Column)
		fmt.Printf("cause: %s\n", pe.Err)
		fmt.Printf("formatted:\n%s\n", pe.FormatError())
	}
	// Output:
	// line: 1
	// column: 7
	// cause: '</' is required
	// formatted:
	// parser error : '</' is required
}

func Example_helium_parse_error_sentinel() {
	// ErrParseError wraps a sentinel error identifying the specific
	// parse failure. Use errors.Is to match against known sentinels
	// through Unwrap.

	_, err := helium.NewParser().Parse(context.Background(), []byte(`<root>`))

	fmt.Println("is ErrLtSlashRequired:", errors.Is(err, helium.ErrLtSlashRequired))
	// Output:
	// is ErrLtSlashRequired: true
}

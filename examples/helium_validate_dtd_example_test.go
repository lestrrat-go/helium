package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_validate_dtd() {
	// ValidateDTD enables DTD-based validation during parsing.
	// When a document violates its DTD constraints the parser
	// returns ErrDTDValidationFailed. Individual errors are
	// delivered to the ErrorHandler configured on the parser.

	ctx := context.Background()

	// Use an ErrorCollector to capture individual validation errors.
	collector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)
	p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)

	// This document declares a #REQUIRED attribute "id" on <doc>,
	// but the instance omits it.
	doc, err := p.Parse(ctx, []byte(`<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`))
	// The parser closes the collector automatically after validation.

	// The document is still returned even when validation fails.
	fmt.Printf("doc returned: %v\n", doc != nil)
	fmt.Printf("validation failed: %v\n", errors.Is(err, helium.ErrDTDValidationFailed))

	// Individual errors are available from the collector.
	fmt.Printf("error count: %d\n", len(collector.Errors()))
	for _, e := range collector.Errors() {
		fmt.Println(e)
	}
	// Output:
	// doc returned: true
	// validation failed: true
	// error count: 1
	// element doc: attribute id is required
}

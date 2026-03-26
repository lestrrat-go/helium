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
	// returns a *DTDValidateError that wraps each individual
	// violation as a separate error, accessible via Unwrap().

	p := helium.NewParser().ValidateDTD(true)

	// This document declares a #REQUIRED attribute "id" on <doc>,
	// but the instance omits it.
	doc, err := p.Parse(context.Background(), []byte(`<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`))

	// The document is still returned even when validation fails.
	fmt.Printf("doc returned: %v\n", doc != nil)

	// Extract the DTDValidateError to inspect individual violations.
	var ve *helium.DTDValidateError
	if errors.As(err, &ve) {
		fmt.Printf("validation errors: %d\n", len(ve.Unwrap()))
		for _, sub := range ve.Unwrap() {
			fmt.Println(sub)
		}
	}
	// Output:
	// doc returned: true
	// validation errors: 1
	// element doc: attribute id is required
}

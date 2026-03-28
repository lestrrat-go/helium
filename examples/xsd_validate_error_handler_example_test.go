package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func Example_xsd_validate_error_handler() {
	// When validating an XML document against a schema, individual
	// validation errors can be received through an ErrorHandler.
	// This is useful when you want to inspect or log each error
	// separately, rather than parsing the combined error string.

	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:all>
        <xs:element name="count" type="xs:integer"/>
        <xs:element name="score" type="xs:decimal"/>
      </xs:all>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	ctx := context.Background()
	p := helium.NewParser()

	schemaDoc, err := p.Parse(ctx, []byte(schemaSrc))
	if err != nil {
		fmt.Printf("failed to parse schema: %s\n", err)
		return
	}
	schema, err := xsd.NewCompiler().Compile(ctx, schemaDoc)
	if err != nil {
		fmt.Printf("failed to compile schema: %s\n", err)
		return
	}

	// This document has two problems:
	//   1. <count> contains "abc" which is not a valid xs:integer.
	//   2. <score> contains "xyz" which is not a valid xs:decimal.
	// Both errors are delivered individually to the ErrorHandler.
	const src = `<root><count>abc</count><score>xyz</score></root>`
	doc, err := p.Parse(ctx, []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Create an ErrorCollector to receive each validation error.
	// ErrorLevelNone collects all errors regardless of severity.
	collector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

	// Create a validator and attach the collector as its ErrorHandler.
	// The validator closes the collector automatically after Validate
	// returns, so collected errors are ready to read immediately.
	v := xsd.NewValidator(schema).
		ErrorHandler(collector)

	_ = v.Validate(ctx, doc)

	// Each validation error is available as a separate entry.
	for i, e := range collector.Errors() {
		// Strip the filename prefix so output is stable across environments.
		msg := e.Error()
		if idx := strings.Index(msg, "Schemas validity error"); idx >= 0 {
			msg = msg[idx:]
		}
		fmt.Printf("error %d: %s", i+1, msg)
	}
	// Output:
	// error 1: Schemas validity error : Element 'count': 'abc' is not a valid value of the atomic type 'xs:integer'.
	// error 2: Schemas validity error : Element 'score': 'xyz' is not a valid value of the atomic type 'xs:decimal'.
}

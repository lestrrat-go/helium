package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

func Example_relaxng_validate_error_handler() {
	// When validation fails, individual errors can be collected via
	// an ErrorHandler.

	ctx := context.Background()
	p := helium.NewParser()

	schemaDoc, err := p.Parse(ctx, []byte(
		`<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="book">
      <element name="title"><text/></element>
    </element>
  </start>
</grammar>`))
	if err != nil {
		fmt.Printf("schema parse failed: %s\n", err)
		return
	}

	grammar, err := relaxng.NewCompiler().Compile(ctx, schemaDoc)
	if err != nil {
		fmt.Printf("schema compile failed: %s\n", err)
		return
	}

	// This document is invalid: <book> requires a <title> child,
	// but only <author> is present.
	doc, err := p.Parse(ctx, []byte(`<book><author>X</author></book>`))
	if err != nil {
		fmt.Printf("xml parse failed: %s\n", err)
		return
	}

	collector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

	v := relaxng.NewValidator(grammar).
		Label("doc.xml").
		ErrorHandler(collector)

	err = v.Validate(ctx, doc)
	fmt.Println("is ErrValidationFailed:", errors.Is(err, relaxng.ErrValidationFailed))

	// Individual errors are available from the collector.
	for _, e := range collector.Errors() {
		fmt.Println(e)
	}
	// Output:
	// is ErrValidationFailed: true
	// doc.xml:1: element book: Relax-NG validity error : Element book failed to validate content
}

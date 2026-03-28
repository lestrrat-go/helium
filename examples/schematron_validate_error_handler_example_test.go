package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
)

func Example_schematron_validate_error_handler() {
	// When validation fails, individual errors can be collected via
	// an ErrorHandler. Schematron delivers structured ValidationError
	// values that can be inspected with errors.AsType.

	ctx := context.Background()
	p := helium.NewParser()

	schemaDoc, err := p.Parse(ctx, []byte(
		`<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern name="book-check">
    <rule context="book">
      <assert test="title">title is required</assert>
    </rule>
  </pattern>
</schema>`))
	if err != nil {
		fmt.Printf("schema parse failed: %s\n", err)
		return
	}

	schema, err := schematron.NewCompiler().Compile(ctx, schemaDoc)
	if err != nil {
		fmt.Printf("schema compile failed: %s\n", err)
		return
	}

	// This document is invalid: the <book> element has no <title>.
	doc, err := p.Parse(ctx, []byte(`<book/>`))
	if err != nil {
		fmt.Printf("xml parse failed: %s\n", err)
		return
	}

	collector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

	v := schematron.NewValidator(schema).
		Label("doc.xml").
		ErrorHandler(collector)

	err = v.Validate(ctx, doc)
	fmt.Println("is ErrValidationFailed:", errors.Is(err, schematron.ErrValidationFailed))

	// Schematron errors implement *schematron.ValidationError with
	// structured fields: Element, Path, and Message.
	for _, e := range collector.Errors() {
		if ve, ok := errors.AsType[*schematron.ValidationError](e); ok {
			fmt.Printf("element: %s\n", ve.Element)
			fmt.Printf("path: %s\n", ve.Path)
			fmt.Printf("message: %s\n", ve.Message)
		}
	}
	// Output:
	// is ErrValidationFailed: true
	// element: book
	// path: /book
	// message: title is required
}

package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func Example_xsd_validate_errors() {
	// This schema requires <root> to have a "version" attribute and
	// contain one or more <item> child elements.
	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:attribute name="version" type="xs:string" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	p := helium.NewParser()

	// Compile the schema from an in-memory document.
	schemaDoc, err := p.Parse(context.Background(), []byte(schemaSrc))
	if err != nil {
		fmt.Printf("failed to parse schema: %s\n", err)
		return
	}
	schema, err := xsd.NewCompiler().Compile(context.Background(), schemaDoc)
	if err != nil {
		fmt.Printf("failed to compile schema: %s\n", err)
		return
	}

	// This document is intentionally invalid: the required "version"
	// attribute is missing from <root>.
	const src = `<root><item>one</item></root>`
	doc, err := p.Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Create a validator from the compiled schema and check the document.
	// Validate returns ErrValidationFailed when the document is invalid.
	v := xsd.NewValidator(schema)

	if err := v.Validate(context.Background(), doc); err != nil {
		fmt.Println(errors.Is(err, xsd.ErrValidationFailed))
	}
	// Output:
	// true
}

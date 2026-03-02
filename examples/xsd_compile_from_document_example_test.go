package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func Example_xsd_compile_from_document() {
	// A simple schema that declares a single <greeting> element of type xs:string.
	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="greeting" type="xs:string"/>
</xs:schema>`

	// Parse the schema source into a DOM document first.
	schemaDoc, err := helium.Parse(context.Background(), []byte(schemaSrc))
	if err != nil {
		fmt.Printf("failed to parse schema: %s\n", err)
		return
	}

	// xsd.Compile compiles a schema from an in-memory Document, as opposed
	// to xsd.CompileFile which reads from a file path. Use Compile when the
	// schema is available as a string or byte slice rather than on disk.
	schema, err := xsd.Compile(schemaDoc)
	if err != nil {
		fmt.Printf("failed to compile schema: %s\n", err)
		return
	}

	// Parse the document to validate.
	doc, err := helium.Parse(context.Background(), []byte(`<greeting>hello</greeting>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Validate the document against the compiled schema.
	if err := xsd.Validate(doc, schema); err != nil {
		fmt.Println(err)
	}
	// Output:
}

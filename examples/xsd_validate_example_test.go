package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func Example_xsd_validate() {
	// Define an XML Schema (XSD) that describes the expected structure:
	//   - <root> element with a required "version" attribute
	//   - <root> contains one or more <item> elements (xs:string)
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

	// Compile parses and compiles the XSD schema from an in-memory document.
	schemaDoc, err := helium.NewParser().Parse(context.Background(), []byte(schemaSrc))
	if err != nil {
		fmt.Printf("failed to parse schema: %s\n", err)
		return
	}
	schema, err := xsd.Compile(context.Background(), schemaDoc)
	if err != nil {
		fmt.Printf("failed to compile schema: %s\n", err)
		return
	}

	// Parse the XML document to validate.
	const src = `<root version="1.0"><item>one</item><item>two</item></root>`
	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Validate checks the document against the compiled schema.
	// It returns nil if the document is valid, or a *ValidateError with details.
	if err := xsd.Validate(context.Background(), doc, schema); err != nil {
		fmt.Println(err)
	}
	// Output:
}

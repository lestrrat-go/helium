package examples_test

import (
	"fmt"
	"strings"

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
	schemaDoc, err := helium.Parse([]byte(schemaSrc))
	if err != nil {
		fmt.Printf("failed to parse schema: %s\n", err)
		return
	}
	schema, err := xsd.Compile(schemaDoc)
	if err != nil {
		fmt.Printf("failed to compile schema: %s\n", err)
		return
	}

	// Parse the XML document to validate.
	const src = `<root version="1.0"><item>one</item><item>two</item></root>`
	doc, err := helium.Parse([]byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Validate checks the document against the compiled schema.
	// The result string contains "validates" if the document is valid,
	// or error messages describing validation failures.
	result := xsd.Validate(doc, schema)
	fmt.Println(strings.Contains(result, "validates"))
	// Output:
	// true
}

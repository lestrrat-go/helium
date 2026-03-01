package examples_test

import (
	"fmt"
	"strings"

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

	// Compile the schema from an in-memory document.
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

	// This document is intentionally invalid: the required "version"
	// attribute is missing from <root>.
	const src = `<root><item>one</item></root>`
	doc, err := helium.Parse([]byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Validate returns a result string. When validation fails, the string
	// contains "fails to validate" along with detailed error messages.
	result := xsd.Validate(doc, schema)
	fmt.Println(strings.Contains(result, "fails to validate"))
	// Output:
	// true
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func Example_xsd_schema_lookup() {
	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:books"
  xmlns:tns="urn:books"
  elementFormDefault="qualified">
  <xs:complexType name="BookType">
    <xs:sequence>
      <xs:element name="title" type="xs:string"/>
    </xs:sequence>
    <xs:attribute name="isbn" type="xs:string" use="required"/>
  </xs:complexType>
  <xs:element name="book" type="tns:BookType"/>
</xs:schema>`

	schemaDoc, err := helium.NewParser().Parse(context.Background(), []byte(schemaSrc))
	if err != nil {
		fmt.Printf("failed to parse schema: %s\n", err)
		return
	}

	// Once a schema is compiled, you can query its global declarations. That is
	// useful for tooling, schema inspection, or building validation workflows
	// that need to discover elements and types programmatically.
	schema, err := xsd.Compile(context.Background(), schemaDoc)
	if err != nil {
		fmt.Printf("failed to compile schema: %s\n", err)
		return
	}

	fmt.Println(schema.TargetNamespace())

	elem, ok := schema.LookupElement("book", "urn:books")
	if !ok {
		fmt.Println("element not found")
		return
	}
	fmt.Printf("%s -> %s\n", elem.Name.Local, elem.Type.Name.Local)

	typ, ok := schema.LookupType("BookType", "urn:books")
	if !ok {
		fmt.Println("type not found")
		return
	}
	fmt.Printf("attrs: %d, element-only: %t\n", len(typ.Attributes), typ.ContentType == xsd.ContentTypeElementOnly)
	// Output:
	// urn:books
	// book -> BookType
	// attrs: 1, element-only: true
}

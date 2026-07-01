# xsd

The `xsd` package compiles XML Schema documents and validates XML instances.

Import path: `github.com/lestrrat-go/helium/xsd`

## Schema version

The compiler targets **XSD 1.0** by default. **XSD 1.1** is opt-in via
`Compiler.Version(xsd.Version11)` (or a `vc:minVersion="1.1"` attribute on the
root `<xs:schema>` when the version is not set explicitly). XSD 1.1 support
includes assertions, conditional type assignment, open content, wildcard
`notNamespace`/`notQName`, `xs:all` relaxations, `xs:override`, document-wide
xs:ID/xs:IDREF/xs:ENTITY value-space validation, and identity-constraint
scoping. It passes the XSD-1.1-tagged test groups of the W3C XML Schema Test
Suite — 967 groups from the IBM, Saxon, Oracle, and W3C-WG collections, 0
failures. The suite's XSD 1.0-era collections (Microsoft, NIST, Sun, Boeing)
contribute no 1.1 tests and are not part of that run.

<!-- INCLUDE(examples/xsd_validate_example_test.go) -->
```go
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

  p := helium.NewParser()

  // Compile parses and compiles the XSD schema from an in-memory document.
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

  // Parse the XML document to validate.
  const src = `<root version="1.0"><item>one</item><item>two</item></root>`
  doc, err := p.Parse(context.Background(), []byte(src))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  // Validate checks the document against the compiled schema. It returns nil
  // if the document is valid, ErrValidationFailed if it is invalid, or
  // ErrNilSchema/ErrNilDocument when the schema or document is nil.
  if err := xsd.NewValidator(schema).Validate(context.Background(), doc); err != nil {
    fmt.Println(err)
  }
  // Output:
}
```
source: [examples/xsd_validate_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xsd_validate_example_test.go)
<!-- END INCLUDE -->

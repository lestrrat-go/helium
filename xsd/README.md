# xsd

The `xsd` package compiles XML Schema documents and validates XML instances.

Import path: `github.com/lestrrat-go/helium/xsd`

## Schema version

The compiler targets **XSD 1.0 by default**. **XSD 1.1 is opt-in** — enable it
with `Compiler.Version(xsd.Version11)`, or add `vc:minVersion="1.1"` to the root
`<xs:schema>` (used only when the version is not set explicitly).

### XSD 1.1 features

Assertions, conditional type assignment, open content, wildcard
`notNamespace`/`notQName`, `xs:all` relaxations, `xs:override`, document-wide
xs:ID/xs:IDREF/xs:ENTITY value-space validation, and identity-constraint scoping.

### Conformance

Measured against the W3C XML Schema Test Suite:

| Mode | Pass | Skip | Fail | Total | Pass/Total | Collections |
|------|-----:|-----:|-----:|------:|-----------:|-------------|
| **XSD 1.0** (default) | 14,314 | 0 | 85 | 14,399 | 99.41% | Microsoft, NIST, Sun, Boeing, IBM, Saxon, Oracle, W3C-WG |
| **XSD 1.1** | 1,046 | 0 | 3 | 1,049 | 99.71% | IBM, Saxon, Oracle, W3C-WG |

The 1.0-era collections (Microsoft, NIST, Sun, Boeing) contain no 1.1-tagged
cases, so they are not part of the 1.1 run.

Committed evidence sits beside this package — `summary-xsd10.md` /
`summary-xsd11.md` and JUnit `results-xsd10.xml` / `results-xsd11.xml` —
regenerated from the sibling `helium-w3c-tests` module:

```sh
go run ./cmd/w3ctest -no-system-out \
  -summary ../helium/xsd/summary-xsd10.md \
  -out ../helium/xsd/results-xsd10.xml xsd10
```

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

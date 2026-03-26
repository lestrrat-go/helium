# schematron

The `schematron` package compiles Schematron schemas and validates XML
documents with XPath-based rules.

Import path: `github.com/lestrrat-go/helium/schematron`

<!-- INCLUDE(examples/schematron_validate_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/schematron"
)

func Example_schematron_validate() {
  p := helium.NewParser()

  // Compile a minimal Schematron schema with one assertion.
  schemaDoc, err := p.Parse(context.Background(), []byte(
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

  schema, err := schematron.NewCompiler().Compile(context.Background(), schemaDoc)
  if err != nil {
    fmt.Printf("schema compile failed: %s\n", err)
    return
  }

  doc, err := p.Parse(context.Background(), []byte(`<book><title>Helium</title></book>`))
  if err != nil {
    fmt.Printf("xml parse failed: %s\n", err)
    return
  }

  if err := schematron.NewValidator(schema).Filename("doc.xml").Validate(context.Background(), doc); err != nil {
    fmt.Println(err)
  }
  // Output:
}
```
source: [examples/schematron_validate_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/schematron_validate_example_test.go)
<!-- END INCLUDE -->

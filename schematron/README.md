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

  // Create a validator from the compiled schema. Label sets the
  // document name used in error messages (it does not read from disk).
  v := schematron.NewValidator(schema).
    Label("doc.xml")

  if err := v.Validate(context.Background(), doc); err != nil {
    fmt.Println(err)
  }
  // Output:
}
```
source: [examples/schematron_validate_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/schematron_validate_example_test.go)
<!-- END INCLUDE -->

## Query language bindings

The `queryBinding` attribute of the `<schema>` element names the query
language the schema's expressions are written in. Two families are
implemented:

| `queryBinding` value | Engine |
|---|---|
| absent, `xslt`, `xslt1`, `xpath`, `xpath1` | XPath 1.0 |
| `xslt3`, `xpath3`, `xpath31` | XPath 3.1 |

Values are matched without regard to case and surrounding whitespace. Any
other value — including the XSLT 2.0 and XPath 2.0 bindings and the non-XPath
bindings the standard mentions (`xquery`, `stx`, `exslt`) — fails compilation
with `ErrUnsupportedQueryBinding`.

`Compiler.QueryBinding(b)` forces a binding and ignores the attribute
entirely. `Compiler.DefaultQueryBinding(b)` chooses the binding for a schema
that names none; a schema that names one still wins.

Under the XPath 3.1 binding a schema cannot reach the filesystem or the
network: `fn:doc`, `fn:collection` and `fn:unparsed-text` need a URI resolver
or an HTTP client, and the evaluator is given neither.

Rule contexts are translated to XPath expressions in both bindings, so a
context that is an XSLT match pattern but not an expression — `key('k','v')`,
for instance — is not supported.

<!-- INCLUDE(examples/schematron_query_binding_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/schematron"
)

func Example_schematron_query_binding() {
  p := helium.NewParser()

  // queryBinding="xslt3" compiles the schema's expressions as XPath 3.1,
  // so the rule may call a function XPath 1.0 does not have.
  schemaDoc, err := p.Parse(context.Background(), []byte(
    `<schema xmlns="http://purl.oclc.org/dsdl/schematron" queryBinding="xslt3">
  <pattern name="isbn-check">
    <rule context="book">
      <assert test="matches(@isbn, '^[0-9]{13}$')">isbn must be 13 digits</assert>
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
  fmt.Printf("binding: %s\n", schema.QueryBinding())

  doc, err := p.Parse(context.Background(), []byte(`<book isbn="12345"/>`))
  if err != nil {
    fmt.Printf("xml parse failed: %s\n", err)
    return
  }

  // The isbn is too short, so the assertion fires and Validate reports a
  // failure. Attach an ErrorHandler to receive the individual messages.
  if err := schematron.NewValidator(schema).Label("book.xml").Validate(context.Background(), doc); err != nil {
    fmt.Println(err)
  }

  // A binding this package does not implement is refused at compile time
  // rather than evaluated as if it were XPath 1.0.
  rejectedDoc, err := p.Parse(context.Background(), []byte(
    `<schema xmlns="http://purl.oclc.org/dsdl/schematron" queryBinding="xslt2">
  <pattern>
    <rule context="book">
      <assert test="@isbn">isbn is required</assert>
    </rule>
  </pattern>
</schema>`))
  if err != nil {
    fmt.Printf("schema parse failed: %s\n", err)
    return
  }

  if _, err := schematron.NewCompiler().Compile(context.Background(), rejectedDoc); err != nil {
    fmt.Println(err)
  }
  // Output:
  // binding: xpath3
  // schematron: validation failed
  // schematron: unsupported query language binding "xslt2" (supported: xslt, xslt1, xpath, xpath1, xslt3, xpath3, xpath31)
}
```
source: [examples/schematron_query_binding_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/schematron_query_binding_example_test.go)
<!-- END INCLUDE -->

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

| Mode | Pass | Skip | XFail | Fail | Total | Pass/Total | Collections |
|------|-----:|-----:|------:|-----:|------:|-----------:|-------------|
| **XSD 1.0** (default) | 14,378 | 0 | 21 | 0 | 14,399 | 99.85% | Microsoft, NIST, Sun, Boeing, IBM, Saxon, Oracle, W3C-WG |
| **XSD 1.1** | 1,048 | 0 | 1 | 0 | 1,049 | 99.90% | IBM, Saxon, Oracle, W3C-WG |

Zero cases fail unexpectedly. The XFail cases are documented expected failures
(`expectations/xsd10.json` and `expectations/xsd11.json` in the sibling
`helium-w3c-tests` module). The 21 XSD 1.0 xfails are W3C tests that are
spec-disputed/queried (bugzilla 4126/4133/4135/4952/4957, w3c/xsdtests#11),
1.0-vs-1.1 particle-restriction subsumption tradeoffs that XSD 1.1 abolished,
libxml2-parity divergences where helium's behaviour is correct, or five IBM
XSD 1.1 wildcard tests whose test group lacks `version="1.1"` and whose schemas
use `notNamespace`/`notQName`, which XSD 1.0 does not define. The one XSD 1.1
xfail, `saxonMeta/Simple.testSet/simple006`, is a disputed §5.3
missing-component case: helium accepts it to match the near-identical
`saxonMeta/Missing/missing006`, which expects valid. Each summary lists the
xfails in its own row with a per-reason breakdown.

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

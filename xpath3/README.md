# xpath3

The `xpath3` package compiles and evaluates XPath 3.1 expressions.

Import path: `github.com/lestrrat-go/helium/xpath3`

## Conformance

The package targets **XPath 3.1** (in XSD 1.1 mode). Against the W3C QT3 test
suite:

| Outcome | Count |
|---------|------:|
| Pass | 22,090 |
| Skip | 383 |
| Fail | 0 |
| Total | 22,473 |

The skips are out-of-scope dependencies — cases requiring an XSLT transform,
XQuery `load-xquery-module`, a schema-validated source, static typing, XPath
2.0-only behavior, XML 1.1, or Unicode 7.0. Committed evidence sits beside this
package — a stamped `summary-qt3.md` and JUnit `results-qt3.xml` — regenerated
from the sibling `helium-w3c-tests` module (`go run ./cmd/w3ctest
-no-system-out -summary ../helium/xpath3/summary-qt3.md -out
../helium/xpath3/results-qt3.xml qt3`).

<!-- INCLUDE(examples/xpath3_find_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_find() {
  doc, err := helium.NewParser().Parse(context.Background(), []byte(`<catalog><book id="1">Go</book><book id="2">XML</book><magazine/></catalog>`))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  // Compile the expression, evaluate it, and extract the matching nodes.
  expr, err := xpath3.NewCompiler().Compile("//book")
  if err != nil {
    fmt.Printf("compile error: %s\n", err)
    return
  }

  r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
    Evaluate(context.Background(), expr, doc)
  if err != nil {
    fmt.Printf("xpath error: %s\n", err)
    return
  }

  nodes, err := r.Nodes()
  if err != nil {
    fmt.Printf("nodes error: %s\n", err)
    return
  }

  fmt.Printf("found %d nodes\n", len(nodes))
  for _, n := range nodes {
    fmt.Printf("  %s: %s\n", n.Name(), string(n.Content()))
  }
  // Output:
  // found 2 nodes
  //   book: Go
  //   book: XML
}
```
source: [examples/xpath3_find_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xpath3_find_example_test.go)
<!-- END INCLUDE -->

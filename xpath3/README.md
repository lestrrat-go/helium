# xpath3

The `xpath3` package compiles and evaluates XPath 3.1 expressions.

Import path: `github.com/lestrrat-go/helium/xpath3`

## Conformance

The package targets **XPath 3.1** (in XSD 1.1 mode). Against the W3C QT3 test
suite:

| Outcome | Count |
|---------|------:|
| Pass | 22,328 |
| Skip | 141 |
| Fail | 0 |
| Total | 22,469 |

The harness really evaluates every FOTS assertion — generic `<assert>`,
`assert-type` (`$result instance of T`), `assert-xml` (serialize + canonical XML
compare), and `assert-permutation` (type-aware atomic value compare + node
`fn:deep-equal`) — and a weak-un-skip guard (`TestQT3WeakNoOpGuard`) fails if a
run-enabled case degrades to a no-op pass. The committed `false_pass_risk` count
is **0**: no case slips through on an unchecked assertion, so a green run reflects
real conformance rather than silent no-op passes.

The 141 skips split into **136 out-of-scope** (XQuery `load-xquery-module`,
static typing, XSD 1.0, XML 1.1, Unicode 7.0, remote HTTP access, and
directory-as-collection URIs) and **5 not-wired** harness gaps (`fn:transform`
fixture-base-URI/resource cases). Zero cases fail.

### Known gaps

The floor is four documented expected-fails, all genuine unimplemented-feature
gaps rather than bugs:

- `json-to-xml-016` — `fn:json-to-xml(..., map{'validate':true()})` needs
  schema-import/PSVI type annotation so `j:number` is typed `xs:double`; helium
  leaves it untyped, so `data(...) instance of xs:double` is false.
- `analyzeString-020`, `analyzeString-021` — the built-in
  `fn:analyze-string-result` schema annotates the constructed result (`@nr` as
  `xs:positiveInteger`, a named complex type on the element); helium leaves it
  untyped without a compiled schema.
- `parse-xml-010` — the parsed document declares and references a `SYSTEM`
  external parsed entity; helium does not fetch and expand it, so the referenced
  content is absent from the result.

The first three are PSVI/schema-awareness gaps; the last is external parsed-entity
expansion.

Committed evidence sits beside this package — a stamped `summary-qt3.md` and
JUnit `results-qt3.xml` — regenerated from the sibling `helium-w3c-tests` module
(`go run ./cmd/w3ctest -no-system-out -summary ../helium/xpath3/summary-qt3.md
-out ../helium/xpath3/results-qt3.xml qt3`).

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

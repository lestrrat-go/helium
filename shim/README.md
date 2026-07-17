# shim

The `shim` package provides a drop-in replacement for Go's `encoding/xml`
package backed by helium's parser.

Import path: `github.com/lestrrat-go/helium/shim`

It exposes the same core API surface as `encoding/xml`, including
`Marshal`, `Unmarshal`, `NewEncoder`, `NewDecoder`, `Token`,
`EncodeToken`, and the familiar struct tags such as `xml:"name,attr"`,
`,chardata`, `,innerxml`, and `,omitempty`.

<!-- INCLUDE(examples/shim_marshal_example_test.go) -->
```go
package examples_test

import (
  "fmt"

  "github.com/lestrrat-go/helium/shim"
)

func Example_shim_marshal() {
  // shim.Marshal works like encoding/xml.Marshal: it serializes a Go
  // struct into XML bytes using struct tags.
  type Person struct {
    XMLName shim.Name `xml:"person"`
    Name    string    `xml:"name"`
    Age     int       `xml:"age"`
  }

  p := Person{Name: "Alice", Age: 30}
  data, err := shim.Marshal(p)
  if err != nil {
    fmt.Printf("error: %s\n", err)
    return
  }
  fmt.Println(string(data))
  // Output:
  // <person><name>Alice</name><age>30</age></person>
}
```
source: [examples/shim_marshal_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/shim_marshal_example_test.go)
<!-- END INCLUDE -->

## Notes

- `Decoder.Strict = false` is not supported.
- `HTMLAutoClose` is omitted and `Decoder.AutoClose` is a no-op.
- Undeclared namespace prefixes are rejected.
- The helium parser is the single authority for the XML declaration. Its parse
  decides the XMLDecl grammar, the version rule, and placement, and shim's
  verdict is helium's; `Unmarshal`, a reader-backed `Decoder`, and a
  TokenReader-backed `Decoder` agree.
- shim accepts the XML versions helium accepts — 1.0 **and** 1.1 (helium
  implements XML 1.1) — where `encoding/xml` rejects `version="1.1"`. A version
  outside the 1.x family (e.g. `2.0`) is rejected. `Unmarshal` and the
  reader-backed `Decoder` accept 1.1 directly; a TokenReader-backed `Decoder`
  accepts a 1.1 declaration once delivered as a token, but an `encoding/xml`
  decoder used as the TokenReader cannot deliver one — it rejects 1.1 during its
  own tokenization, a limitation of `encoding/xml`, not shim.
- An XML declaration that does not conform to the XMLDecl grammar is rejected
  by every entry point: a `charset=` pseudo-attribute, a missing or empty
  version, an empty encoding, a `standalone` that is not `yes`/`no`, a repeated
  pseudo-attribute, or pseudo-attributes out of order. `encoding/xml` accepts
  them all.
- An XML declaration is admitted only as the very first thing in the document,
  with only whitespace allowed ahead of it. Both `Unmarshal` and `Decoder`
  reject a `<?xml` following an earlier declaration, a comment, a processing
  instruction, or a doctype; `encoding/xml` accepts it as an ordinary
  `ProcInst`.
- The target `xml` is reserved in any casing (`PITarget ::= Name -
  (('X'|'x')('M'|'m')('L'|'l'))`), so `<?XML ...?>`, `<?Xml ...?>` and
  `<?xMl ...?>` are illegal wherever they appear and are rejected by every entry
  point. A target that merely begins with `xml` (`<?xmlversion ="2.0"?>`,
  `<?xml-stylesheet ...?>`) is an ordinary PI, not a declaration, and is
  accepted.
- A declaration with whitespace around the version pseudo-attribute's `=`
  (`<?xml version = "2.0"?>`) is rejected as an unsupported version;
  `encoding/xml` accepts it.
- Namespace declarations are emitted before regular attributes.
- `InputOffset` is approximate rather than exact.
- Empty elements in `,innerxml` may serialize as self-closed tags.

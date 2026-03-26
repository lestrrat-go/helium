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
- Namespace declarations are emitted before regular attributes.
- `InputOffset` is approximate rather than exact.
- Empty elements in `,innerxml` may serialize as self-closed tags.

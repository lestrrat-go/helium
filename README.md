# helium

[![Build Status](https://travis-ci.org/lestrrat-go/helium.svg?branch=main)](https://travis-ci.org/lestrrat-go/helium)
[![GoDoc](https://godoc.org/github.com/lestrrat-go/helium?status.svg)](https://godoc.org/github.com/lestrrat-go/helium)

Pure-Go replacement of libxml2. Initially by hand, and later with the help of AI agents.

Attempts to do everything libxml2 can, while making it more idiomatic Go where it makes sense to do so.

# SYNOPSIS

<!-- INCLUDE(examples/helium_parse_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
)

func Example_helium_parse() {
  // helium.Parse is the simplest way to parse an XML document from a byte slice.
  // It returns a *helium.Document representing the parsed DOM tree.
  doc, err := helium.Parse(context.Background(), []byte(`<root><child>hello</child></root>`))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  // XMLString serializes the entire document back to an XML string,
  // including the XML declaration (<?xml version="1.0"?>).
  s, err := doc.XMLString()
  if err != nil {
    fmt.Printf("failed to serialize: %s\n", err)
    return
  }
  fmt.Println(s)
  // Output:
  // <?xml version="1.0"?>
  // <root><child>hello</child></root>
}
```
source: [examples/helium_parse_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/helium_parse_example_test.go)
<!-- END INCLUDE -->

# SAX2

<!-- INCLUDE(examples/sax_parse_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/sax"
)

func Example_sax_parse() {
  const src = `<library><book lang="en">Go Programming</book><book lang="ja">Goプログラミング</book></library>`

  // sax.New creates a SAX handler with all callbacks set to nil (no-ops).
  // You only need to set the callbacks you care about.
  handler := sax.New()

  // OnStartElementNS is called when an opening tag is encountered.
  // It receives the local name, prefix, namespace URI, any namespace
  // declarations, and the element's attributes.
  //
  // The handler field expects a sax.StartElementNS interface, so we wrap
  // the function literal with sax.StartElementNSFunc to satisfy it.
  handler.OnStartElementNS = sax.StartElementNSFunc(func(_ sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
    fmt.Printf("<%s", localname)
    for _, a := range attrs {
      fmt.Printf(" %s=%q", a.Name(), a.Value())
    }
    fmt.Print(">")
    return nil
  })

  // OnEndElementNS is called when a closing tag is encountered.
  handler.OnEndElementNS = sax.EndElementNSFunc(func(_ sax.Context, localname, prefix, uri string) error {
    fmt.Printf("</%s>\n", localname)
    return nil
  })

  // OnCharacters is called for text content between tags.
  handler.OnCharacters = sax.CharactersFunc(func(_ sax.Context, ch []byte) error {
    fmt.Print(string(ch))
    return nil
  })

  // Attach the SAX handler to a parser. When a SAX handler is set,
  // the parser fires events instead of building a full DOM tree.
  p := helium.NewParser()
  p.SetSAXHandler(handler)

  // Parse triggers the SAX events. The returned document may be nil
  // or minimal when using SAX mode, since the purpose is event-driven
  // processing rather than DOM construction.
  _, err := p.Parse(context.Background(), []byte(src))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }
  // Output:
  // <library><book lang="en">Go Programming</book>
  // <book lang="ja">Goプログラミング</book>
  // </library>
}
```
source: [examples/sax_parse_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/sax_parse_example_test.go)
<!-- END INCLUDE -->

# HTML

<!-- INCLUDE(examples/html_parse_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/html"
  "github.com/lestrrat-go/helium/xpath"
)

func Example_html_parse() {
  // html.Parse builds a helium DOM from HTML input and applies HTML-specific
  // parsing rules (implied elements, case-insensitive tag handling, etc.).
  doc, err := html.Parse(context.Background(), []byte(`<h1>Title</h1><div>Hello</div>`))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  // The parsed document uses the HTML document node type.
  fmt.Println(doc.Type() == helium.HTMLDocumentNode)

  // Parsed HTML can be queried with regular XPath helpers.
  nodes, err := xpath.Find(doc, `//div`)
  if err != nil {
    fmt.Printf("xpath failed: %s\n", err)
    return
  }
  fmt.Println(len(nodes))
  fmt.Println(string(nodes[0].Content()))
  // Output:
  // true
  // 1
  // Hello
}
```
source: [examples/html_parse_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/html_parse_example_test.go)
<!-- END INCLUDE -->

# XPath

<!-- INCLUDE(examples/xpath_find_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/xpath"
)

func Example_xpath_find() {
  doc, err := helium.Parse(context.Background(), []byte(`<catalog><book id="1">Go</book><book id="2">XML</book><magazine/></catalog>`))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  // xpath.Find is a convenience function that evaluates an XPath expression
  // and returns the resulting node set directly. It is a shorthand for
  // calling Evaluate and accessing the NodeSet field of the result.
  // The expression "//book" selects all <book> elements anywhere in the
  // document tree.
  nodes, err := xpath.Find(doc, "//book")
  if err != nil {
    fmt.Printf("xpath error: %s\n", err)
    return
  }

  fmt.Printf("found %d nodes\n", len(nodes))
  for _, n := range nodes {
    // Name returns the element's local name, and Content returns
    // the concatenated text content of the element and its descendants.
    fmt.Printf("  %s: %s\n", n.Name(), string(n.Content()))
  }
  // Output:
  // found 2 nodes
  //   book: Go
  //   book: XML
}
```
source: [examples/xpath_find_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xpath_find_example_test.go)
<!-- END INCLUDE -->

# XInclude

<!-- INCLUDE(examples/xinclude_process_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"
  "os"
  "path/filepath"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/xinclude"
)

func Example_xinclude_process() {
  // XInclude allows XML documents to include content from other XML files.
  // The main document references an external fragment via <xi:include>.
  const mainSrc = `<?xml version="1.0"?>
<doc xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="fragment.xml"/>
</doc>`

  // This is the content of the included fragment.
  const fragmentSrc = `<?xml version="1.0"?>
<included>hello from fragment</included>`

  // Create a temporary directory and write both files to it.
  // The parser needs real files on disk because XInclude resolves
  // hrefs relative to the base URI of the including document.
  dir, err := os.MkdirTemp(".", ".tmp-xinclude-*")
  if err != nil {
    fmt.Printf("failed to create temp dir: %s\n", err)
    return
  }
  defer os.RemoveAll(dir) //nolint:errcheck

  // Convert to absolute path so the XInclude processor can correctly
  // resolve relative hrefs without path-doubling issues.
  absDir, err := filepath.Abs(dir)
  if err != nil {
    fmt.Printf("failed to get abs path: %s\n", err)
    return
  }

  mainPath := filepath.Join(absDir, "main.xml")
  fragPath := filepath.Join(absDir, "fragment.xml")
  if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
    fmt.Printf("failed to write: %s\n", err)
    return
  }
  if err := os.WriteFile(fragPath, []byte(fragmentSrc), 0644); err != nil {
    fmt.Printf("failed to write: %s\n", err)
    return
  }

  // parseMain is a helper that parses the main document from disk.
  // SetBaseURI tells the parser the file's location so relative hrefs
  // in xi:include can be resolved.
  parseMain := func() (*helium.Document, error) {
    data, err := os.ReadFile(mainPath)
    if err != nil {
      return nil, err
    }
    p := helium.NewParser()
    p.SetBaseURI(mainPath)
    return p.Parse(context.Background(), data)
  }

  // --- Default behavior: marker nodes ---
  //
  // By default (matching libxml2's behavior), xinclude.Process replaces
  // each xi:include element with a pair of XIncludeStart/XIncludeEnd
  // marker nodes that bracket the included content. These markers
  // serialize as empty <xi:include> elements, allowing applications to
  // track which parts of the tree were included.
  doc, err := parseMain()
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  n, err := xinclude.Process(doc,
    xinclude.WithBaseURI(mainPath),
    xinclude.WithNoBaseFixup(),
  )
  if err != nil {
    fmt.Printf("xinclude error: %s\n", err)
    return
  }
  fmt.Printf("substitutions: %d\n", n)

  s, err := doc.XMLString()
  if err != nil {
    fmt.Printf("failed to serialize: %s\n", err)
    return
  }
  fmt.Printf("with markers:\n%s", s)

  // --- WithNoXIncludeMarkers: clean output ---
  //
  // WithNoXIncludeMarkers (equivalent to libxml2's XML_PARSE_NOXINCNODE)
  // removes the xi:include elements entirely after substitution,
  // leaving only the included content in the tree.
  doc, err = parseMain()
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  n, err = xinclude.Process(doc,
    xinclude.WithBaseURI(mainPath),
    xinclude.WithNoBaseFixup(),
    xinclude.WithNoXIncludeMarkers(),
  )
  if err != nil {
    fmt.Printf("xinclude error: %s\n", err)
    return
  }
  fmt.Printf("substitutions: %d\n", n)

  s, err = doc.XMLString()
  if err != nil {
    fmt.Printf("failed to serialize: %s\n", err)
    return
  }
  fmt.Printf("without markers:\n%s", s)
  // Output:
  // substitutions: 1
  // with markers:
  // <?xml version="1.0"?>
  // <doc xmlns:xi="http://www.w3.org/2001/XInclude">
  //   <xi:include></xi:include><included>hello from fragment</included><xi:include></xi:include>
  // </doc>
  // substitutions: 1
  // without markers:
  // <?xml version="1.0"?>
  // <doc xmlns:xi="http://www.w3.org/2001/XInclude">
  //   <included>hello from fragment</included>
  // </doc>
}
```
source: [examples/xinclude_process_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/xinclude_process_example_test.go)
<!-- END INCLUDE -->

# C14N

<!-- INCLUDE(examples/c14n_canonicalize_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/c14n"
)

func Example_c14n_canonicalize() {
  // In the source, attributes are in order b="2", a="1".
  // C14N (Canonical XML) sorts attributes lexicographically,
  // so the canonical form will have a="1" before b="2".
  const src = `<root b="2" a="1"><child/></root>`

  doc, err := helium.Parse(context.Background(), []byte(src))
  if err != nil {
    fmt.Printf("failed to parse: %s\n", err)
    return
  }

  // CanonicalizeTo serializes the document in canonical form and returns
  // the result as a byte slice. C14N10 selects the Canonical XML 1.0
  // algorithm (https://www.w3.org/TR/xml-c14n).
  //
  // Key properties of canonical form:
  //   - No XML declaration
  //   - Attributes sorted by namespace URI then local name
  //   - Empty elements use start-tag + end-tag (not self-closing)
  //   - Whitespace in attribute values is normalized
  out, err := c14n.CanonicalizeTo(doc, c14n.C14N10)
  if err != nil {
    fmt.Printf("failed to canonicalize: %s\n", err)
    return
  }
  fmt.Print(string(out))
  // Output:
  // <root a="1" b="2"><child></child></root>
}
```
source: [examples/c14n_canonicalize_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/c14n_canonicalize_example_test.go)
<!-- END INCLUDE -->

# RelaxNG

<!-- INCLUDE(examples/relaxng_validate_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"

  "github.com/lestrrat-go/helium"
  "github.com/lestrrat-go/helium/relaxng"
)

func Example_relaxng_validate() {
  // Compile a small RELAX NG schema from XML syntax.
  schemaDoc, err := helium.Parse(context.Background(), []byte(
    `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="book">
      <element name="title"><text/></element>
    </element>
  </start>
</grammar>`))
  if err != nil {
    fmt.Printf("schema parse failed: %s\n", err)
    return
  }

  grammar, err := relaxng.Compile(schemaDoc)
  if err != nil {
    fmt.Printf("schema compile failed: %s\n", err)
    return
  }

  doc, err := helium.Parse(context.Background(), []byte(`<book><title>Helium</title></book>`))
  if err != nil {
    fmt.Printf("xml parse failed: %s\n", err)
    return
  }

  if err := relaxng.Validate(doc, grammar, relaxng.WithFilename("doc.xml")); err != nil {
    fmt.Println(err)
  }
  // Output:
}
```
source: [examples/relaxng_validate_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/relaxng_validate_example_test.go)
<!-- END INCLUDE -->

# Schematron

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
  // Compile a minimal Schematron schema with one assertion.
  schemaDoc, err := helium.Parse(context.Background(), []byte(
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

  schema, err := schematron.Compile(schemaDoc)
  if err != nil {
    fmt.Printf("schema compile failed: %s\n", err)
    return
  }

  doc, err := helium.Parse(context.Background(), []byte(`<book><title>Helium</title></book>`))
  if err != nil {
    fmt.Printf("xml parse failed: %s\n", err)
    return
  }

  if err := schematron.Validate(doc, schema, schematron.WithFilename("doc.xml")); err != nil {
    fmt.Println(err)
  }
  // Output:
}
```
source: [examples/schematron_validate_example_test.go](https://github.com/lestrrat-go/helium/blob/main/examples/schematron_validate_example_test.go)
<!-- END INCLUDE -->

# `heliumlint`

```
heliumlint xmlfile ...
```

```
cat xmlfile | heliumlint
```

# Current status

* Core functionality is implemented: XML/HTML parsing, DOM building, SAX2, XPath, XInclude, C14N, RelaxNG, Schematron, and XSD.
* The codebase includes broad compatibility tests and examples, and active parity work against libxml2 behavior.
* Some edge cases and parity gaps are still being iterated on; contributions and issue reports are welcome.

# Contributing

## Issues

For bug reports and feature requests, please follow the issue template when possible.
If you can include a minimal reproduction or failing test case, that helps a lot.

## Pull Requests

Please include tests that cover your changes.

If your change touches generated files, update the generator/source first, regenerate,
and commit both the source and generated outputs together.

Please keep pull requests focused and small enough to review quickly.

## Discussions / Usage

For usage questions, design discussion, or "is this approach reasonable?" questions,
please open a GitHub Discussion first.


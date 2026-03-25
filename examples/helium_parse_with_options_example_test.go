package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse_with_options() {
	// This XML source contains whitespace-only text nodes (the newline + spaces
	// between <root> and <child>, and between </child> and </root>).
	const src = `<root>
  <child>text</child>
</root>`

	// Create a parser instance to configure parsing options.
	// helium.NewParser() returns a reusable parser that can be customized
	// before calling Parse.
	// NoBlanks tells the parser to discard whitespace-only text nodes.
	// This is useful when you want a compact DOM without insignificant whitespace.
	p := helium.NewParser().NoBlanks(true)

	doc, err := p.Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// The output will be compact (no whitespace between elements)
	// because ParseNoBlanks stripped the whitespace-only text nodes.
	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <?xml version="1.0"?>
	// <root><child>text</child></root>
}

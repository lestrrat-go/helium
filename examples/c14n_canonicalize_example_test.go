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

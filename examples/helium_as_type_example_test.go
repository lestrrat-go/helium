package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_as_type() {
	// AsType safely narrows a Node to a concrete type, similar to
	// errors.AsType. It returns the typed value and true on success,
	// or the zero value and false otherwise.

	doc, err := helium.NewParser().Parse(context.Background(),
		[]byte(`<root><child>hello</child></root>`))
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// The document's first child is the root element.
	node := doc.FirstChild()

	if elem, ok := helium.AsType[*helium.Element](node); ok {
		fmt.Printf("element: %s\n", elem.LocalName())
	}

	// A non-matching assertion returns false.
	if _, ok := helium.AsType[*helium.Text](node); !ok {
		fmt.Println("not a text node")
	}
	// Output:
	// element: root
	// not a text node
}

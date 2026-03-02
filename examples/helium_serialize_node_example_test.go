package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_serialize_node() {
	doc, err := helium.Parse(context.Background(), []byte(`<root><a>1</a><b>2</b></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Navigate to the first child element of <root>.
	// doc.FirstChild() returns <root>, and root.FirstChild() returns <a>.
	first := doc.FirstChild().FirstChild()

	// Type-assert the Node to *helium.Element so we can call XMLString on it.
	// XMLString on an element serializes only that element and its subtree,
	// without the XML declaration — unlike doc.XMLString() which includes it.
	elem := first.(*helium.Element)

	s, err := elem.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <a>1</a>
}

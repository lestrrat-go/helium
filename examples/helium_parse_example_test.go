package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse() {
	// helium.Parse is the simplest way to parse an XML document from a byte slice.
	// It returns a *helium.Document representing the parsed DOM tree.
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><child>hello</child></root>`))
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

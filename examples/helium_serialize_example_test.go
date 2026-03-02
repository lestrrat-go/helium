package examples_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_serialize() {
	doc, err := helium.Parse(context.Background(), []byte(`<root><child>text</child></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// XMLString returns the entire XML document as a string.
	// This is convenient for small documents or when you need a string value.
	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(s)

	// XML writes the document to any io.Writer, which is more efficient
	// for large documents or when streaming to a file/network connection.
	var buf bytes.Buffer
	if err := doc.XML(&buf); err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(buf.String())
	// Output:
	// <?xml version="1.0"?>
	// <root><child>text</child></root>
	// <?xml version="1.0"?>
	// <root><child>text</child></root>
}

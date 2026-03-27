package examples_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_serialize() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><child>text</child></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// WriteString returns the entire XML document as a string.
	// This is convenient for small documents or when you need a string value.
	s, err := helium.WriteString(doc)
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(s)

	// Write writes the document to any io.Writer, which is more efficient
	// for large documents or when streaming to a file/network connection.
	var buf bytes.Buffer
	if err := helium.Write(&buf, doc); err != nil {
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

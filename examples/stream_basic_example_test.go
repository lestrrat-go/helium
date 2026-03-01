package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_basic() {
	// stream provides a streaming, forward-only XML writer that writes
	// directly to an io.Writer. This is efficient for generating large
	// XML documents because it does not build an in-memory DOM tree.
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)

	// StartDocument writes the XML declaration.
	// Arguments: version, encoding, standalone.
	// An empty encoding or standalone string omits that attribute.
	if err := w.StartDocument("1.0", "UTF-8", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// StartElement opens a new element tag. You must close it later
	// with EndElement.
	if err := w.StartElement("greeting"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteString writes text content inside the current element.
	// Special characters (<, >, &, etc.) are automatically escaped.
	if err := w.WriteString("hello world"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// EndElement closes the most recently opened element.
	if err := w.EndElement(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// EndDocument flushes any remaining state and finalizes the document.
	if err := w.EndDocument(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <?xml version="1.0" encoding="UTF-8"?>
	// <greeting>hello world</greeting>
}

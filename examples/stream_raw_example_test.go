package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_raw() {
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)

	if err := w.StartElement("root"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteRaw injects a raw string directly into the output without
	// any XML escaping. This is useful when you have pre-formed XML
	// content (e.g., from another serializer or a template) that you
	// want to embed verbatim.
	//
	// WARNING: The caller is responsible for ensuring the raw content
	// is well-formed XML. The writer performs no validation on raw content.
	if err := w.WriteRaw("<already-escaped attr=\"val\"/>"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := w.EndElement(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <root><already-escaped attr="val"/></root>
}

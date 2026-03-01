package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_namespaces() {
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)

	if err := w.StartDocument("1.0", "", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// StartElementNS opens a namespaced element. Arguments are:
	//   prefix    - the namespace prefix (e.g., "soap")
	//   localname - the element's local name (e.g., "Envelope")
	//   uri       - the namespace URI
	// The writer automatically emits the xmlns:prefix="uri" declaration
	// on the first element that uses this prefix.
	if err := w.StartElementNS("soap", "Envelope", "http://schemas.xmlsoap.org/soap/envelope/"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// Using the same prefix and URI again does not re-emit the declaration,
	// since it's already in scope from the parent element.
	if err := w.StartElementNS("soap", "Body", "http://schemas.xmlsoap.org/soap/envelope/"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteElementNS is a convenience method that writes a complete
	// namespaced element (open tag + content + close tag) in one call.
	// Here we introduce a new prefix "m" for a different namespace.
	if err := w.WriteElementNS("m", "GetPrice", "http://example.com/prices", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// EndElement closes <soap:Body>.
	if err := w.EndElement(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// EndDocument closes any remaining open elements (<soap:Envelope>)
	// and flushes the writer.
	if err := w.EndDocument(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <?xml version="1.0"?>
	// <soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><m:GetPrice xmlns:m="http://example.com/prices"></m:GetPrice></soap:Body></soap:Envelope>
}

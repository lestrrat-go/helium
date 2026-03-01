package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_convenience() {
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)

	if err := w.StartElement("person"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteAttribute is a convenience method that writes a complete
	// attribute (name="value") in one call. It is equivalent to:
	//   w.StartAttribute("name")
	//   w.WriteString("Alice")
	//   w.EndAttribute()
	// Attributes must be written immediately after StartElement,
	// before any child content or child elements.
	if err := w.WriteAttribute("name", "Alice"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	if err := w.WriteAttribute("age", "30"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteElement is a convenience method that writes a complete
	// element (start tag + text content + end tag) in one call.
	// It is equivalent to:
	//   w.StartElement("email")
	//   w.WriteString("alice@example.com")
	//   w.EndElement()
	if err := w.WriteElement("email", "alice@example.com"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := w.EndElement(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <person name="Alice" age="30"><email>alice@example.com</email></person>
}

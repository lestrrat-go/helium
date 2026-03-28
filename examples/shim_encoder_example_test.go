package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_encoder() {
	// shim.NewEncoder creates a streaming XML encoder that writes to
	// an io.Writer, just like encoding/xml.NewEncoder.
	type Book struct {
		XMLName shim.Name `xml:"book"`
		Title   string    `xml:"title"`
		Author  string    `xml:"author"`
	}

	var buf bytes.Buffer
	enc := shim.NewEncoder(&buf)

	// Indent configures pretty-printing. The first argument is the prefix
	// prepended to each line; the second is the per-level indent string.
	enc.Indent("", "  ")

	b := Book{Title: "The Go Programming Language", Author: "Donovan & Kernighan"}
	if err := enc.Encode(b); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	if err := enc.Close(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Println(buf.String())
	// Output:
	// <book>
	//   <title>The Go Programming Language</title>
	//   <author>Donovan &amp; Kernighan</author>
	// </book>
}

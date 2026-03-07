package examples_test

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_decoder() {
	// shim.NewDecoder creates a streaming XML decoder backed by helium's
	// SAX parser, compatible with encoding/xml.NewDecoder.
	input := `<catalog><book>Go in Action</book><book>The Go Programming Language</book></catalog>`
	dec := shim.NewDecoder(strings.NewReader(input))

	var catalog struct {
		XMLName shim.Name `xml:"catalog"`
		Books   []string  `xml:"book"`
	}

	if err := dec.Decode(&catalog); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	for _, book := range catalog.Books {
		fmt.Println(book)
	}
	// Output:
	// Go in Action
	// The Go Programming Language
}

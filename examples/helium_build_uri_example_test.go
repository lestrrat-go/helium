package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_build_uri() {
	// BuildURI resolves a relative URI against a base URI.
	// If the first argument is already absolute, it is returned as-is.
	abs := helium.BuildURI("schema.xsd", "http://example.com/schemas/main.xsd")
	fmt.Println(abs)

	// An already-absolute URI is returned unchanged.
	abs = helium.BuildURI("http://other.com/data.xml", "http://example.com/base/")
	fmt.Println(abs)
	// Output:
	// http://example.com/schemas/schema.xsd
	// http://other.com/data.xml
}

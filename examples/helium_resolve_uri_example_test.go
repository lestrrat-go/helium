package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_resolve_uri() {
	// ResolveURI uses the conventional (base, reference) argument order, matching
	// url.URL.ResolveReference and RFC 3986. It wraps the libxml2-parity BuildURI
	// (which takes its arguments in the reverse order).
	resolved, err := helium.ResolveURI("http://example.com/schemas/main.xsd", "schema.xsd")
	if err != nil {
		fmt.Printf("failed to resolve reference: %s\n", err)
		return
	}
	fmt.Println(resolved)

	// An already-absolute reference is returned unchanged.
	resolved, err = helium.ResolveURI("http://example.com/base/", "http://other.com/data.xml")
	if err != nil {
		fmt.Printf("failed to resolve reference: %s\n", err)
		return
	}
	fmt.Println(resolved)
	// Output:
	// http://example.com/schemas/schema.xsd
	// http://other.com/data.xml
}

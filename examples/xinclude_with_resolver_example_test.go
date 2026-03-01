package examples_test

import (
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
)

// memoryResolver resolves xi:include hrefs from an in-memory map
// instead of reading from the filesystem. This is useful for testing
// or when XML fragments are generated dynamically.
type memoryResolver struct {
	files map[string]string
}

// Resolve implements the xinclude.Resolver interface. It receives the
// href from the xi:include element and the base URI of the including
// document. It returns an io.ReadCloser with the resolved content.
func (r *memoryResolver) Resolve(href, base string) (io.ReadCloser, error) {
	content, ok := r.files[href]
	if !ok {
		return nil, fmt.Errorf("not found: %s", href)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func Example_xinclude_with_resolver() {
	// The document contains an xi:include that references "greeting.xml".
	const src = `<doc xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="greeting.xml"/></doc>`

	doc, err := helium.Parse([]byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Create a custom resolver that serves content from memory.
	resolver := &memoryResolver{
		files: map[string]string{
			"greeting.xml": `<greeting>hello</greeting>`,
		},
	}

	// Process the XInclude directives with our custom resolver.
	//
	// WithResolver supplies the custom Resolver implementation.
	// WithNoXIncludeNodes removes the xi:include namespace declaration
	//   from the output after processing.
	// WithNoBaseFixup prevents adding xml:base attributes to included content.
	n, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
		xinclude.WithNoBaseFixup(),
	)
	if err != nil {
		fmt.Printf("xinclude error: %s\n", err)
		return
	}
	fmt.Printf("substitutions: %d\n", n)

	// The xi:include element has been replaced with the <greeting> element.
	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(s)
	// Output:
	// substitutions: 1
	// <?xml version="1.0"?>
	// <doc xmlns:xi="http://www.w3.org/2001/XInclude"><greeting>hello</greeting></doc>
}

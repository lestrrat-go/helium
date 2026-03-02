package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
)

func Example_sax_count_elements() {
	const src = `<root><a><b/><b/></a><c/><a><b/></a></root>`

	// This example demonstrates using SAX for streaming analysis.
	// We count occurrences of each element without building a DOM tree,
	// which is memory-efficient for large documents.
	counts := map[string]int{}

	handler := sax.New()

	// Only set OnStartElementNS — we don't need end-element or
	// character callbacks for simple counting.
	// Wrap the function with sax.StartElementNSFunc to satisfy the
	// sax.StartElementNS interface expected by the handler field.
	handler.OnStartElementNS = sax.StartElementNSFunc(func(_ sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		counts[localname]++
		return nil
	})

	p := helium.NewParser()
	p.SetSAXHandler(handler)

	_, err := p.Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Print counts in a deterministic order for the test output.
	fmt.Printf("root: %d\n", counts["root"])
	fmt.Printf("a: %d\n", counts["a"])
	fmt.Printf("b: %d\n", counts["b"])
	fmt.Printf("c: %d\n", counts["c"])
	// Output:
	// root: 1
	// a: 2
	// b: 3
	// c: 1
}

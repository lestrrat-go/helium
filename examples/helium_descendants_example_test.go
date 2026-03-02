package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_descendants() {
	doc, err := helium.Parse(context.Background(), []byte(`<a><b><c/></b><d/></a>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Descendants performs a depth-first pre-order traversal of all
	// descendants, excluding the starting node itself.
	// Do not modify the tree (add/remove/reorder nodes) during iteration.
	for d := range helium.Descendants(doc.DocumentElement()) {
		if d.Type() == helium.ElementNode {
			fmt.Println(d.Name())
		}
	}
	// Output:
	// b
	// c
	// d
}

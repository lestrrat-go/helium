package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_walk() {
	doc, err := helium.Parse(context.Background(), []byte(`<a><b><c/></b><d/></a>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Walk performs a depth-first traversal of the entire document tree.
	// The callback is invoked for every node (document, element, text, etc.).
	// Here we filter for element nodes only and print their names.
	// The traversal order is: a, b, c, d (depth-first).
	err = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() == helium.ElementNode {
			fmt.Println(n.Name())
		}
		return nil
	})
	if err != nil {
		fmt.Printf("walk error: %s\n", err)
		return
	}
	// Output:
	// a
	// b
	// c
	// d
}

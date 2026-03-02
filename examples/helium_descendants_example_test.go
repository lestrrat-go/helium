package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_descendants() {
	doc, err := helium.Parse([]byte(`<a><b><c/></b><d/></a>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Descendants performs a depth-first pre-order traversal of all
	// descendants, excluding the starting node itself.
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

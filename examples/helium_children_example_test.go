package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_children() {
	doc, err := helium.Parse([]byte(`<root><a/><b/><c/></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Children iterates over the direct children of a node.
	for child := range helium.Children(doc.DocumentElement()) {
		fmt.Println(child.Name())
	}
	// Output:
	// a
	// b
	// c
}

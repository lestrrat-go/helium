package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_node_get_base() {
	doc, err := helium.Parse([]byte(`<root xml:base="http://example.com/a/"><child xml:base="b/"/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	child := doc.DocumentElement().FirstChild()
	fmt.Println(helium.NodeGetBase(doc, child))
	// Output:
	// http://example.com/a/b/
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_node_get_base() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root xml:base="http://example.com/a/"><child xml:base="b/"/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	// NodeGetBase resolves the effective base URI for a node by walking
	// xml:base declarations up the ancestor chain and resolving relative values.
	child := doc.DocumentElement().FirstChild()
	fmt.Println(helium.NodeGetBase(doc, child))
	// Output:
	// http://example.com/a/b/
}

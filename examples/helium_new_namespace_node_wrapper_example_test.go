package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_new_namespace_node_wrapper() {
	doc, err := helium.Parse(context.Background(), []byte(`<root/>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	// Namespace declarations are not ordinary element nodes. When an API expects
	// a helium.Node, NewNamespaceNodeWrapper lets you expose a namespace binding
	// through that interface while still keeping its owning element context.
	ns := helium.NewNamespace("p", "urn:p")
	wrapped := helium.NewNamespaceNodeWrapper(ns, doc.DocumentElement())

	fmt.Println(wrapped.Name())
	fmt.Println(string(wrapped.Content()))
	fmt.Println(wrapped.Parent().Name())
	// Output:
	// p
	// urn:p
	// root
}

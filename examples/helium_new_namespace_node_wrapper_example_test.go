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

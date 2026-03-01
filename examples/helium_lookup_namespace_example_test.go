package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_lookup_namespace() {
	doc, err := helium.Parse([]byte(`<root xmlns:a="urn:a"><child/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}
	root := doc.DocumentElement()

	byPrefix := helium.LookupNSByPrefix(root, "a")
	byHref := helium.LookupNSByHref(root, "urn:a")
	custom := helium.NewNamespace("b", "urn:b")

	fmt.Println(byPrefix.Prefix())
	fmt.Println(byHref.URI())
	fmt.Printf("%s %s\n", custom.Prefix(), custom.URI())
	// Output:
	// a
	// urn:a
	// b urn:b
}

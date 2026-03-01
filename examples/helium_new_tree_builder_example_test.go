package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_new_tree_builder() {
	tb := helium.NewTreeBuilder()
	p := helium.NewParser()
	p.SetSAXHandler(tb)

	doc, err := p.Parse([]byte(`<root><item>ok</item></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	fmt.Println(doc.DocumentElement().FirstChild().Name())
	// Output:
	// item
}

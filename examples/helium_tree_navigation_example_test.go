package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_tree_navigation() {
	doc, err := helium.Parse([]byte(`<root><a/><b/><c/></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// FirstChild of a Document returns the root (document) element.
	root := doc.FirstChild()
	fmt.Printf("root: %s\n", root.Name())

	// Iterate over an element's children using the FirstChild/NextSibling pattern.
	// This is the standard linked-list traversal used throughout the DOM.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		fmt.Printf("child: %s\n", child.Name())
	}

	// Parent returns the parent node. Here, the parent of <a> is <root>.
	first := root.FirstChild()
	fmt.Printf("parent of %s: %s\n", first.Name(), first.Parent().Name())
	// Output:
	// root: root
	// child: a
	// child: b
	// child: c
	// parent of a: root
}

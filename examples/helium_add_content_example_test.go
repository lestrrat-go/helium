package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_add_content() {
	doc := helium.CreateDocument()

	root, err := doc.CreateElement("msg")
	if err != nil {
		fmt.Printf("failed to create element: %s\n", err)
		return
	}
	if err := doc.SetDocumentElement(root); err != nil {
		fmt.Printf("failed to set root: %s\n", err)
		return
	}

	// AddContent appends text to the element. If the last child is already
	// a text node, the new content is merged into it (concatenated),
	// rather than creating a separate text node. This means two consecutive
	// AddContent calls produce a single text node "hello world".
	if err := root.AddContent([]byte("hello ")); err != nil {
		fmt.Printf("failed to add content: %s\n", err)
		return
	}
	if err := root.AddContent([]byte("world")); err != nil {
		fmt.Printf("failed to add content: %s\n", err)
		return
	}

	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <?xml version="1.0"?>
	// <msg>hello world</msg>
}

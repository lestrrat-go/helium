package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_create_document() {
	// NewDocument creates a new XML document with the specified version,
	// encoding, and standalone declaration. StandaloneExplicitNo produces
	// standalone="no" in the XML declaration.
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)

	// CreateElement creates a new element node owned by this document.
	// The element is not yet attached to the tree.
	root := doc.CreateElement("catalog")

	// SetDocumentElement attaches the element as the root (document element).
	if err := doc.SetDocumentElement(root); err != nil {
		fmt.Printf("failed to set root: %s\n", err)
		return
	}

	// Create a child element and set an attribute on it.
	book := doc.CreateElement("book")
	if _, err := book.SetAttribute("id", "b1"); err != nil {
		fmt.Printf("failed to set attribute: %s\n", err)
		return
	}

	// AddChild appends the element as the last child of the parent.
	if err := root.AddChild(book); err != nil {
		fmt.Printf("failed to add child: %s\n", err)
		return
	}

	// Build a nested structure: catalog > book > title
	title := doc.CreateElement("title")

	// AppendText creates a text node with the given bytes and
	// appends it as a child of the element.
	if err := title.AppendText([]byte("XML in Practice")); err != nil {
		fmt.Printf("failed to add content: %s\n", err)
		return
	}
	if err := book.AddChild(title); err != nil {
		fmt.Printf("failed to add child: %s\n", err)
		return
	}

	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <?xml version="1.0" encoding="UTF-8" standalone="no"?>
	// <catalog><book id="b1"><title>XML in Practice</title></book></catalog>
}

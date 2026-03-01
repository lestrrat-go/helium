package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_create_namespaced() {
	// CreateDocument creates a minimal document with just an XML declaration
	// (version="1.0", no encoding, no standalone).
	doc := helium.CreateDocument()

	root, err := doc.CreateElement("feed")
	if err != nil {
		fmt.Printf("failed to create element: %s\n", err)
		return
	}
	if err := doc.SetDocumentElement(root); err != nil {
		fmt.Printf("failed to set root: %s\n", err)
		return
	}

	// SetNamespace declares a namespace on the element.
	// The first argument is the prefix (empty string "" means default namespace).
	// The second argument is the namespace URI.
	// This produces xmlns="http://www.w3.org/2005/Atom" on the element.
	if err := root.SetNamespace("", "http://www.w3.org/2005/Atom"); err != nil {
		fmt.Printf("failed to set namespace: %s\n", err)
		return
	}

	// Child elements inherit the default namespace from their parent.
	entry, err := doc.CreateElement("entry")
	if err != nil {
		fmt.Printf("failed to create element: %s\n", err)
		return
	}
	if err := root.AddChild(entry); err != nil {
		fmt.Printf("failed to add child: %s\n", err)
		return
	}

	title, err := doc.CreateElement("title")
	if err != nil {
		fmt.Printf("failed to create element: %s\n", err)
		return
	}
	if err := title.AddContent([]byte("Example")); err != nil {
		fmt.Printf("failed to add content: %s\n", err)
		return
	}
	if err := entry.AddChild(title); err != nil {
		fmt.Printf("failed to add child: %s\n", err)
		return
	}

	// The xmlns declaration only appears on the root element;
	// child elements are implicitly in the same namespace.
	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <?xml version="1.0"?>
	// <feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Example</title></entry></feed>
}

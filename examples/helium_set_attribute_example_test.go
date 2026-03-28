package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_set_attribute() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root><item/></root>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// Navigate to the <item> element and type-assert to *helium.Element.
	item := doc.FirstChild().FirstChild().(*helium.Element)

	// SetAttribute adds or replaces an attribute on the element.
	// Attributes appear in the order they are set.
	if _, err := item.SetAttribute("id", "42"); err != nil {
		fmt.Printf("failed to set attribute: %s\n", err)
		return
	}
	if _, err := item.SetAttribute("class", "active"); err != nil {
		fmt.Printf("failed to set attribute: %s\n", err)
		return
	}

	s, err := helium.WriteString(doc)
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <?xml version="1.0"?>
	// <root><item id="42" class="active"/></root>
}

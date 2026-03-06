package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_marshal_attr() {
	// Struct fields tagged with ",attr" are serialized as XML attributes,
	// matching encoding/xml behavior.
	type Link struct {
		XMLName shim.Name `xml:"a"`
		Href    string    `xml:"href,attr"`
		Text    string    `xml:",chardata"`
	}

	link := Link{Href: "https://example.com", Text: "Example"}
	data, err := shim.Marshal(link)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Println(string(data))
	// Output:
	// <a href="https://example.com">Example</a>
}

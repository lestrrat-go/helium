package examples_test

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/html"
)

func Example_html_parse_with_sax() {
	// ParseWithSAX emits streaming callbacks without constructing a DOM tree.
	var sawH1 bool
	var sawTitle bool

	handler := &html.SAXCallbacks{}
	handler.StartElementHandler = html.StartElementFunc(func(name string, _ []html.Attribute) error {
		if strings.EqualFold(name, "h1") {
			sawH1 = true
		}
		return nil
	})
	handler.CharactersHandler = html.CharactersFunc(func(ch []byte) error {
		if strings.TrimSpace(string(ch)) == "Title" {
			sawTitle = true
		}
		return nil
	})

	if err := html.ParseWithSAX([]byte(`<h1>Title</h1>`), handler); err != nil {
		fmt.Printf("failed to parse with SAX: %s\n", err)
		return
	}

	fmt.Println(sawH1)
	fmt.Println(sawTitle)
	// Output:
	// true
	// true
}

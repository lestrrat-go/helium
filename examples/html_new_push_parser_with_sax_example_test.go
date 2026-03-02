package examples_test

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/html"
)

func Example_html_new_push_parser_with_sax() {
	var sawTitle bool
	handler := &html.SAXCallbacks{}
	handler.OnStartElement = html.StartElementFunc(func(name string, _ []html.Attribute) error {
		if strings.EqualFold(name, "h1") {
			sawTitle = true
		}
		return nil
	})

	pp := html.NewPushParserWithSAX(handler)
	if err := pp.Push([]byte(`<h1>Title</h1>`)); err != nil {
		fmt.Printf("push failed: %s\n", err)
		return
	}
	if _, err := pp.Close(); err != nil {
		fmt.Printf("close failed: %s\n", err)
		return
	}

	fmt.Println(sawTitle)
	// Output:
	// true
}

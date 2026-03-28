package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/html"
)

func Example_html_new_sax_push_parser() {
	var sawTitle bool
	handler := &html.SAXCallbacks{}
	handler.SetOnStartElement(html.StartElementFunc(func(name string, _ []html.Attribute) error {
		if strings.EqualFold(name, "h1") {
			sawTitle = true
		}
		return nil
	}))

	// NewSAXPushParser combines push parsing with SAX-style event delivery.
	// Each Push call fires SAX callbacks as elements are encountered.
	p := html.NewParser()
	pp := p.NewSAXPushParser(context.Background(), handler)
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

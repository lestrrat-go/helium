package examples_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/html"
)

func Example_html_write_doc() {
	// html.NewWriter().WriteTo serializes an entire HTML document using HTML
	// output rules. That means you get document-level behavior such as an HTML
	// doctype and HTML-friendly serialization, rather than raw XML formatting.
	doc, err := html.NewParser().Parse(context.Background(), []byte(`<html><body><div>Hello</div></body></html>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	var buf bytes.Buffer
	if err := html.NewWriter().WriteTo(&buf, doc); err != nil {
		fmt.Printf("failed to write: %s\n", err)
		return
	}

	// The exact serialized string may contain more HTML boilerplate than the
	// fragment you started with, so the example checks for the important pieces.
	out := buf.String()
	fmt.Println(strings.HasPrefix(out, "<!DOCTYPE html PUBLIC"))
	fmt.Println(strings.Contains(out, "<div>Hello</div>"))
	// Output:
	// true
	// true
}

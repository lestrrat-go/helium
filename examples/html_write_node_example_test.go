package examples_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/xpath1"
)

func Example_html_write_node() {
	// html.WriteNode is the fragment-oriented counterpart to html.WriteDoc.
	// Use it when you already selected a subtree and only want that node's HTML.
	doc, err := html.Parse(context.Background(), []byte(`<div><p>Hello</p></div>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	nodes, err := xpath1.Find(context.Background(), doc, `//p`)
	if err != nil {
		fmt.Printf("xpath failed: %s\n", err)
		return
	}

	var buf bytes.Buffer
	if err := html.WriteNode(&buf, nodes[0]); err != nil {
		fmt.Printf("write failed: %s\n", err)
		return
	}

	// This is intentionally checking a fragment, not a full HTML document.
	fmt.Println(strings.Contains(buf.String(), "<p>Hello</p>"))
	// Output:
	// true
}

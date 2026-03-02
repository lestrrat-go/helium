package examples_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_html_write_node() {
	doc, err := html.Parse([]byte(`<div><p>Hello</p></div>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	nodes, err := xpath.Find(doc, `//p`)
	if err != nil {
		fmt.Printf("xpath failed: %s\n", err)
		return
	}

	var buf bytes.Buffer
	if err := html.WriteNode(&buf, nodes[0]); err != nil {
		fmt.Printf("write failed: %s\n", err)
		return
	}

	fmt.Println(strings.Contains(buf.String(), "<p>Hello</p>"))
	// Output:
	// true
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/xpath1"
)

func Example_html_new_push_parser() {
	// HTML push parser accepts input in chunks, just like the XML push
	// parser. The HTML parser is more lenient — unclosed tags, missing
	// end tags, etc. are recovered automatically.
	p := html.NewParser()
	pp := p.NewPushParser()
	if err := pp.Push([]byte(`<h1>Title`)); err != nil {
		fmt.Printf("push failed: %s\n", err)
		return
	}
	if err := pp.Push([]byte(`</h1>`)); err != nil {
		fmt.Printf("push failed: %s\n", err)
		return
	}

	doc, err := pp.Flush(context.Background())
	if err != nil {
		fmt.Printf("flush failed: %s\n", err)
		return
	}

	nodes, err := xpath1.Find(context.Background(), doc, `//h1`)
	if err != nil {
		fmt.Printf("xpath failed: %s\n", err)
		return
	}

	fmt.Println(string(nodes[0].Content()))
	// Output:
	// Title
}

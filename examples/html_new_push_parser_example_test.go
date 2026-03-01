package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_html_new_push_parser() {
	pp := html.NewPushParser()
	if err := pp.Push([]byte(`<h1>Title`)); err != nil {
		fmt.Printf("push failed: %s\n", err)
		return
	}
	if err := pp.Push([]byte(`</h1>`)); err != nil {
		fmt.Printf("push failed: %s\n", err)
		return
	}

	doc, err := pp.Close()
	if err != nil {
		fmt.Printf("close failed: %s\n", err)
		return
	}

	nodes, err := xpath.Find(doc, `//h1`)
	if err != nil {
		fmt.Printf("xpath failed: %s\n", err)
		return
	}

	fmt.Println(string(nodes[0].Content()))
	// Output:
	// Title
}

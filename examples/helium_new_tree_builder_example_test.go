package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_new_tree_builder() {
	// NewTreeBuilder turns SAX parser events back into a regular helium DOM.
	// Use it when you want parser-level control (custom SAX handlers, push
	// parsing, parser options) but still want a finished *helium.Document.
	tb := helium.NewTreeBuilder()

	// SAXHandler installs the tree builder as the parser's SAX event
	// receiver. The parser fires events; the tree builder reassembles
	// them into a DOM tree.
	p := helium.NewParser().
		SAXHandler(tb)

	// Because the tree builder is installed as the SAX handler, Parse still
	// returns a document even though the parser is running in SAX mode.
	doc, err := p.Parse(context.Background(), []byte(`<root><item>ok</item></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	fmt.Println(doc.DocumentElement().FirstChild().Name())
	// Output:
	// item
}

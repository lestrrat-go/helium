package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpointer"
)

func Example_xpointer_evaluate() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<doc><chapter><section>first</section></chapter><chapter><section>second</section></chapter></doc>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// XPointer's element() scheme navigates the document tree by child position.
	// The path "/1/2/1" means:
	//   /1 → first child element of the document → <doc>
	//   /2 → second child of <doc>               → <chapter> (the second one)
	//   /1 → first child of that <chapter>        → <section>
	//
	// Child indices in element() are 1-based and only count element nodes
	// (text nodes, comments, etc. are skipped).
	nodes, err := xpointer.Evaluate(context.Background(), doc, "element(/1/2/1)")
	if err != nil {
		fmt.Printf("xpointer error: %s\n", err)
		return
	}

	for _, n := range nodes {
		fmt.Printf("%s: %s\n", n.Name(), string(n.Content()))
	}
	// Output:
	// section: second
}

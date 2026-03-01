package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
)

func Example_c14n_with_comments() {
	// This document contains a comment node between <root> and <child>.
	const src = `<root><!-- a comment --><child/></root>`

	doc, err := helium.Parse([]byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// By default, C14N strips comment nodes from the output.
	without, err := c14n.CanonicalizeTo(doc, c14n.C14N10)
	if err != nil {
		fmt.Printf("failed: %s\n", err)
		return
	}
	fmt.Println(string(without))

	// WithComments() preserves comment nodes in the canonical output.
	// The C14N spec defines two variants for each algorithm: one that
	// includes comments and one that omits them. WithComments() selects
	// the "with comments" variant.
	with, err := c14n.CanonicalizeTo(doc, c14n.C14N10, c14n.WithComments())
	if err != nil {
		fmt.Printf("failed: %s\n", err)
		return
	}
	fmt.Print(string(with))
	// Output:
	// <root><child></child></root>
	// <root><!-- a comment --><child></child></root>
}

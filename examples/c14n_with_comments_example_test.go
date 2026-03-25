package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
)

func Example_c14n_with_comments() {
	// This document contains a comment node between <root> and <child>.
	const src = `<root><!-- a comment --><child/></root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// By default, C14N strips comment nodes from the output.
	without, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	if err != nil {
		fmt.Printf("failed: %s\n", err)
		return
	}
	fmt.Println(string(without))

	// Comments() preserves comment nodes in the canonical output.
	// The C14N spec defines two variants for each algorithm: one that
	// includes comments and one that omits them. Comments() selects
	// the "with comments" variant.
	with, err := c14n.NewCanonicalizer(c14n.C14N10).Comments().CanonicalizeTo(doc)
	if err != nil {
		fmt.Printf("failed: %s\n", err)
		return
	}
	fmt.Print(string(with))
	// Output:
	// <root><child></child></root>
	// <root><!-- a comment --><child></child></root>
}

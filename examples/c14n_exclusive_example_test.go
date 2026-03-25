package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
)

func Example_c14n_exclusive() {
	// This document declares three namespace prefixes: a, b, and c.
	// However, only prefix "a" is actually used (on <a:item/>).
	// Prefixes "b" and "c" are declared but never used in element
	// or attribute names.
	const src = `<root xmlns:a="http://a" xmlns:b="http://b"><child xmlns:c="http://c"><a:item/></child></root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// ExclusiveC14N10 implements Exclusive Canonical XML 1.0
	// (https://www.w3.org/TR/xml-exc-c14n/). Unlike regular C14N which
	// inherits all ancestor namespace declarations, exclusive C14N only
	// renders namespace declarations that are "visibly utilized" by the
	// element or its attributes. This makes it suitable for canonicalizing
	// XML fragments (e.g., in XML digital signatures) without pulling in
	// unrelated namespace context.
	//
	// In this example, only xmlns:a appears in the output because only
	// the "a" prefix is visibly used (on <a:item>). The "b" and "c"
	// declarations are omitted.
	out, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).CanonicalizeTo(doc)
	if err != nil {
		fmt.Printf("failed to canonicalize: %s\n", err)
		return
	}
	fmt.Print(string(out))
	// Output:
	// <root><child><a:item xmlns:a="http://a"></a:item></child></root>
}

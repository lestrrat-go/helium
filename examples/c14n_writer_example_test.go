package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
)

func Example_c14n_canonicalize_writer() {
	doc, err := helium.Parse([]byte(`<root b="2" a="1"><child/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	var buf bytes.Buffer
	if err := c14n.Canonicalize(&buf, doc, c14n.C14N10); err != nil {
		fmt.Printf("canonicalize failed: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <root a="1" b="2"><child></child></root>
}

package examples_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
)

func Example_c14n_canonicalize_writer() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root b="2" a="1"><child/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	// Canonicalize writes directly to an io.Writer. Use this form when you want
	// to stream canonical XML to a file, network connection, or hash function
	// without building an intermediate byte slice first.
	var buf bytes.Buffer
	if err := c14n.NewCanonicalizer(c14n.C14N10).Canonicalize(doc, &buf); err != nil {
		fmt.Printf("canonicalize failed: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <root a="1" b="2"><child></child></root>
}

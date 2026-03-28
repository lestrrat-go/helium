package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse_reader() {
	// ParseReader accepts an io.Reader instead of a byte slice.
	// Use it when your XML comes from a file, network connection,
	// or any other streaming source.
	p := helium.NewParser()
	doc, err := p.ParseReader(context.Background(), strings.NewReader(`<root><child>ok</child></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	fmt.Println(doc.DocumentElement().FirstChild().Name())
	// Output:
	// child
}

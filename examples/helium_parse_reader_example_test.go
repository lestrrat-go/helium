package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse_reader() {
	doc, err := helium.ParseReader(context.Background(), strings.NewReader(`<root><child>ok</child></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	fmt.Println(doc.DocumentElement().FirstChild().Name())
	// Output:
	// child
}

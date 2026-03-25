package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse_in_node_context() {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root xmlns:ns="urn:ns"><existing/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	n, err := helium.NewParser().ParseInNodeContext(context.Background(), doc.DocumentElement(), []byte(`<ns:item>ok</ns:item>`))
	if err != nil {
		fmt.Printf("parse in context failed: %s\n", err)
		return
	}

	fmt.Println(n.Name())
	fmt.Println(string(n.FirstChild().Content()))
	// Output:
	// ns:item
	// ok
}

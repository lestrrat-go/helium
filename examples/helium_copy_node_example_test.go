package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_copy_node() {
	src, err := helium.Parse(context.Background(), []byte(`<root><item>hello</item></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	// CopyNode performs a deep copy into the target document. Use it when you
	// need to move or duplicate content across document boundaries, since nodes
	// belong to the document that created them.
	item := src.DocumentElement().FirstChild()

	dst := helium.NewDefaultDocument()
	copied, err := helium.CopyNode(item, dst)
	if err != nil {
		fmt.Printf("copy failed: %s\n", err)
		return
	}

	fmt.Println(copied.Name())
	fmt.Println(string(copied.FirstChild().Content()))
	// Output:
	// item
	// hello
}

package examples_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_unlink_node() {
	doc, err := helium.Parse(context.Background(), []byte(`<root><a/><b/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	root := doc.DocumentElement()
	a := root.FirstChild()
	helium.UnlinkNode(a)

	var buf bytes.Buffer
	d := helium.NewWriter()
	if err := d.WriteNode(&buf, root); err != nil {
		fmt.Printf("write failed: %s\n", err)
		return
	}

	fmt.Println(root.FirstChild().Name())
	fmt.Println(a.Parent() == nil)
	// Output:
	// b
	// true
}

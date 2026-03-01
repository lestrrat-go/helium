package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_unlink_node() {
	doc, err := helium.Parse([]byte(`<root><a/><b/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	root := doc.DocumentElement()
	a := root.FirstChild()
	helium.UnlinkNode(a)

	var buf bytes.Buffer
	var d helium.Dumper
	if err := d.DumpNode(&buf, root); err != nil {
		fmt.Printf("dump failed: %s\n", err)
		return
	}

	fmt.Println(root.FirstChild().Name())
	fmt.Println(a.Parent() == nil)
	// Output:
	// b
	// true
}

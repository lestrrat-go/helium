package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_copy_node() {
	src, err := helium.Parse([]byte(`<root><item>hello</item></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}
	item := src.DocumentElement().FirstChild()

	dst := helium.CreateDocument()
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

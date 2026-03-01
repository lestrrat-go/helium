package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpointer"
)

func Example_xpointer_copy_node() {
	src, err := helium.Parse([]byte(`<doc><section>intro</section></doc>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	dst := helium.CreateDocument()
	copied, err := xpointer.CopyNode(src.DocumentElement().FirstChild(), dst)
	if err != nil {
		fmt.Printf("copy failed: %s\n", err)
		return
	}

	fmt.Println(copied.Name())
	fmt.Println(string(copied.FirstChild().Content()))
	// Output:
	// section
	// intro
}

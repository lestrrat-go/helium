package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_xpointer_copy_node() {
	src, err := helium.NewParser().Parse(context.Background(), []byte(`<doc><section>intro</section></doc>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	dst := helium.NewDefaultDocument()
	copied, err := helium.CopyNode(src.DocumentElement().FirstChild(), dst)
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

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_copy_doc() {
	src, err := helium.NewParser().Parse(context.Background(), []byte(`<root><child>hello</child></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	cloned, err := helium.CopyDoc(src)
	if err != nil {
		fmt.Printf("copy failed: %s\n", err)
		return
	}

	fmt.Println(src == cloned)
	fmt.Println(cloned.DocumentElement().Name())
	// Output:
	// false
	// root
}

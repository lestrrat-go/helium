package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_new_xinclude_marker() {
	doc := helium.NewDefaultDocument()
	n := helium.NewXIncludeMarker(doc, helium.XIncludeStartNode, "include")
	fmt.Println(n.Type() == helium.XIncludeStartNode)
	fmt.Println(n.Name())
	// Output:
	// true
	// include
}

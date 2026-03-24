package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_new_xinclude_marker() {
	doc := helium.NewDefaultDocument()

	// XInclude markers are special internal nodes used to mark include
	// boundaries. Most applications will not create them manually, but they are
	// useful when you need to build or inspect trees that model XInclude state.
	n := helium.NewXIncludeMarker(doc, helium.XIncludeStartNode, "include")
	fmt.Println(n.Type() == helium.XIncludeStartNode)
	fmt.Println(n.Name())
	// Output:
	// true
	// include
}

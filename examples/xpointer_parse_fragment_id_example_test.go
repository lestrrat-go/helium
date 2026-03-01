package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/xpointer"
)

func Example_xpointer_parse_fragment_id() {
	scheme, body, err := xpointer.ParseFragmentID("xpointer(//section[1])")
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}
	fmt.Printf("%s %s\n", scheme, body)

	scheme, body, err = xpointer.ParseFragmentID("intro")
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}
	fmt.Printf("%q %s\n", scheme, body)
	// Output:
	// xpointer //section[1]
	// "" intro
}

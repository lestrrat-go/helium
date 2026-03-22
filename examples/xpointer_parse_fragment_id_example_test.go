package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/xpointer"
)

func Example_xpointer_parse_fragment_id() {
	// ParseFragmentID splits an XPointer fragment into its scheme and body.
	// That is useful when you need to inspect or route fragment identifiers
	// before evaluating them.
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

	// A plain shorthand fragment has no explicit scheme, so the returned scheme
	// string is empty.
	fmt.Printf("%q %s\n", scheme, body)
	// Output:
	// xpointer //section[1]
	// "" intro
}

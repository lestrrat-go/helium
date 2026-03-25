package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_writer_options() {
	const src = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (item,empty)>
  <!ELEMENT item (#PCDATA)>
  <!ELEMENT empty EMPTY>
]>
<root><item>hello</item><empty/></root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	s, err := doc.XMLString(
		helium.WithNoDecl(),
		helium.WithSkipDTD(),
		helium.WithNoEmpty(),
		helium.WithFormat(),
		helium.WithIndentString("\t"),
	)
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(s)
	// Output:
	// <root>
	// 	<item>hello</item>
	// 	<empty></empty>
	// </root>
}

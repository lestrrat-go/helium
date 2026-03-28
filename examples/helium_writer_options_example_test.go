package examples_test

import (
	"context"
	"fmt"
	"strings"

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

	// Configure a writer with several output options. Each method returns
	// a new writer copy, so you can build variants from a shared base.
	var buf strings.Builder
	w := helium.NewWriter().
		XMLDeclaration(false).         // omit the <?xml ...?> declaration
		IncludeDTD(false).             // omit the <!DOCTYPE ...> block
		SelfCloseEmptyElements(false). // write <empty></empty> instead of <empty/>
		Format(true).                  // enable pretty-printing with indentation
		IndentString("\t")             // indent with tabs instead of spaces

	if err := w.WriteTo(&buf, doc); err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Print(buf.String())
	// Output:
	// <root>
	// 	<item>hello</item>
	// 	<empty></empty>
	// </root>
}

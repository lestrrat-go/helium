package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_indent() {
	var buf bytes.Buffer

	// WithIndent enables pretty-printing with the specified indentation string.
	// Each nested level is indented by one copy of this string.
	// Here we use two spaces per level.
	w := stream.NewWriter(&buf, stream.WithIndent("  "))

	if err := w.StartDocument("1.0", "", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := w.StartElement("root"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteElement is a convenience method: start tag + text + end tag.
	// With indentation enabled, each child element is placed on its own
	// line with the appropriate indent level.
	for _, v := range []string{"one", "two"} {
		if err := w.WriteElement("child", v); err != nil {
			fmt.Printf("error: %s\n", err)
			return
		}
	}

	// EndDocument closes <root> and flushes.
	// The closing </root> tag appears at indent level 0.
	if err := w.EndDocument(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <?xml version="1.0"?>
	// <root>
	//   <child>one</child>
	//   <child>two</child>
	// </root>
}

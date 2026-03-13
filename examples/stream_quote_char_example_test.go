package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_quote_char() {
	var buf bytes.Buffer

	w := stream.NewWriter(&buf, stream.WithQuoteChar('\''))
	if err := w.StartDocument("1.0", "", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	if err := w.StartElement("item"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	if err := w.WriteAttribute("name", `Tom "TJ"`); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	if err := w.WriteString("value"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	if err := w.EndDocument(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <?xml version='1.0'?>
	// <item name='Tom "TJ"'>value</item>
}

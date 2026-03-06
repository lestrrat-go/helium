package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_encode_token() {
	// EncodeToken allows writing individual XML tokens to the stream,
	// giving fine-grained control over the output. This mirrors
	// encoding/xml.Encoder.EncodeToken.
	var buf bytes.Buffer
	enc := shim.NewEncoder(&buf)

	// Write a processing instruction
	if err := enc.EncodeToken(shim.ProcInst{Target: "xml", Inst: []byte(`version="1.0"`)}); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// Write a start element with an attribute
	start := shim.StartElement{
		Name: shim.Name{Local: "greeting"},
		Attr: []shim.Attr{
			{Name: shim.Name{Local: "lang"}, Value: "en"},
		},
	}
	if err := enc.EncodeToken(start); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// Write character data
	if err := enc.EncodeToken(shim.CharData("Hello, World!")); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// Write the end element
	if err := enc.EncodeToken(start.End()); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := enc.Flush(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Println(buf.String())
	// Output:
	// <?xml version="1.0"?><greeting lang="en">Hello, World!</greeting>
}

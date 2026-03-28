package examples_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium/shim"
)

func Example_shim_decoder_token() {
	// Decoder.Token() reads individual XML tokens with namespace
	// resolution, similar to encoding/xml.Decoder.Token().
	input := `<root><child>text</child></root>`
	dec := shim.NewDecoder(context.Background(), strings.NewReader(input))

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("error: %s\n", err)
			return
		}
		switch v := tok.(type) {
		case shim.StartElement:
			fmt.Printf("start: %s\n", v.Name.Local)
		case shim.EndElement:
			fmt.Printf("end: %s\n", v.Name.Local)
		case shim.CharData:
			fmt.Printf("text: %s\n", string(v))
		}
	}
	// Output:
	// start: root
	// start: child
	// text: text
	// end: child
	// end: root
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
)

func Example_helium_parser_char_buffer_size() {
	handler := sax.New()
	handler.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
		fmt.Printf("%q\n", ch)
		return nil
	}))

	p := helium.NewParser()
	p.SetSAXHandler(handler)

	// SetCharBufferSize controls how much character data the parser batches into
	// each SAX characters() callback. Smaller buffers can reduce latency for
	// streaming consumers, but callers must be ready to receive one logical text
	// node in multiple chunks.
	p.SetCharBufferSize(3)

	if _, err := p.Parse(context.Background(), []byte(`<root>abcdefg</root>`)); err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}
	// Output:
	// "abc"
	// "def"
	// "g"
}

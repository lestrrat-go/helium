package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
)

func Example_sax_parse() {
	const src = `<library><book lang="en">Go Programming</book><book lang="ja">Goプログラミング</book></library>`

	// sax.New creates a SAX handler with all callbacks set to nil (no-ops).
	// You only need to set the callbacks you care about.
	handler := sax.New()

	// OnStartElementNS is called when an opening tag is encountered.
	// It receives the local name, prefix, namespace URI, any namespace
	// declarations, and the element's attributes.
	//
	// The handler field expects a sax.StartElementNS interface, so we wrap
	// the function literal with sax.StartElementNSFunc to satisfy it.
	handler.OnStartElementNS = sax.StartElementNSFunc(func(_ sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		fmt.Printf("<%s", localname)
		for _, a := range attrs {
			fmt.Printf(" %s=%q", a.Name(), a.Value())
		}
		fmt.Print(">")
		return nil
	})

	// OnEndElementNS is called when a closing tag is encountered.
	handler.OnEndElementNS = sax.EndElementNSFunc(func(_ sax.Context, localname, prefix, uri string) error {
		fmt.Printf("</%s>\n", localname)
		return nil
	})

	// OnCharacters is called for text content between tags.
	handler.OnCharacters = sax.CharactersFunc(func(_ sax.Context, ch []byte) error {
		fmt.Print(string(ch))
		return nil
	})

	// Attach the SAX handler to a parser. When a SAX handler is set,
	// the parser fires events instead of building a full DOM tree.
	p := helium.NewParser()
	p.SetSAXHandler(handler)

	// Parse triggers the SAX events. The returned document may be nil
	// or minimal when using SAX mode, since the purpose is event-driven
	// processing rather than DOM construction.
	_, err := p.Parse([]byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}
	// Output:
	// <library><book lang="en">Go Programming</book>
	// <book lang="ja">Goプログラミング</book>
	// </library>
}

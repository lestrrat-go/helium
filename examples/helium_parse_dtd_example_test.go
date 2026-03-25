package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse_dtd() {
	// This XML document includes an internal DTD subset that declares:
	//   - Element "root" contains one "child" element
	//   - Element "child" contains parsed character data (#PCDATA)
	//   - Element "child" has a default attribute "lang" with value "en"
	//
	// Note that the <child> element in the document body does NOT explicitly
	// set the "lang" attribute — it will be added by the parser when
	// DTD attribute defaulting is enabled.
	const src = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (child)>
  <!ELEMENT child (#PCDATA)>
  <!ATTLIST child lang CDATA "en">
]>
<root><child>hello</child></root>`

	// DTDAttr tells the parser to apply default attribute values
	// defined in the DTD. Without this option, the "lang" attribute
	// would not appear on the <child> element.
	p := helium.NewParser().DTDAttr(true)

	doc, err := p.Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	// The serialized output shows lang="en" on <child>, even though
	// it was not present in the original source — the parser applied
	// the DTD default.
	s, err := doc.XMLString()
	if err != nil {
		fmt.Printf("failed to serialize: %s\n", err)
		return
	}
	fmt.Println(s)
	// Output:
	// <?xml version="1.0"?>
	// <!DOCTYPE root [
	// <!ELEMENT root (child)>
	// <!ELEMENT child (#PCDATA)>
	// <!ATTLIST child lang CDATA "en">
	// ]>
	// <root><child lang="en">hello</child></root>
}

package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_dtd() {
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)

	if err := w.StartDocument("1.0", "", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// StartDTD begins an internal DTD subset: <!DOCTYPE name [...]>
	// Arguments: root element name, publicID, systemID.
	// Empty publicID/systemID omits those identifiers from the DOCTYPE.
	if err := w.StartDTD("note", "", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteDTDElement declares an element's content model.
	// "note" contains a sequence of "to" then "body" child elements.
	if err := w.WriteDTDElement("note", "(to,body)"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	// "to" and "body" contain parsed character data (text).
	if err := w.WriteDTDElement("to", "(#PCDATA)"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	if err := w.WriteDTDElement("body", "(#PCDATA)"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteDTDAttlist declares attribute(s) for an element.
	// Here we declare a "priority" attribute with a default value of "normal".
	if err := w.WriteDTDAttlist("note", `priority CDATA "normal"`); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteDTDEntity declares a general entity. The first argument indicates
	// whether this is a parameter entity (true) or a general entity (false).
	// Here "copy" is defined as the character reference for the copyright symbol.
	if err := w.WriteDTDEntity(false, "copy", "&#169;"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// EndDTD closes the internal DTD subset.
	if err := w.EndDTD(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// Write the root element after the DTD.
	if err := w.WriteElement("note", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := w.EndDocument(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <?xml version="1.0"?>
	// <!DOCTYPE note [<!ELEMENT note (to,body)><!ELEMENT to (#PCDATA)><!ELEMENT body (#PCDATA)><!ATTLIST note priority CDATA "normal"><!ENTITY copy "&amp;#169;">]><note></note>
}

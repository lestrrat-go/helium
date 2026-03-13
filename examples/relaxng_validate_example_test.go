package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

func Example_relaxng_validate() {
	// Compile a small RELAX NG schema from XML syntax.
	schemaDoc, err := helium.Parse(context.Background(), []byte(
		`<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="book">
      <element name="title"><text/></element>
    </element>
  </start>
</grammar>`))
	if err != nil {
		fmt.Printf("schema parse failed: %s\n", err)
		return
	}

	grammar, err := relaxng.Compile(context.Background(), schemaDoc)
	if err != nil {
		fmt.Printf("schema compile failed: %s\n", err)
		return
	}

	doc, err := helium.Parse(context.Background(), []byte(`<book><title>Helium</title></book>`))
	if err != nil {
		fmt.Printf("xml parse failed: %s\n", err)
		return
	}

	if err := relaxng.Validate(doc, grammar, relaxng.WithFilename("doc.xml")); err != nil {
		fmt.Println(err)
	}
	// Output:
}

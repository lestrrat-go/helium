package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

func Example_relaxng_validate() {
	p := helium.NewParser()

	// Compile a small RELAX NG schema from XML syntax.
	schemaDoc, err := p.Parse(context.Background(), []byte(
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

	grammar, err := relaxng.NewCompiler().Compile(context.Background(), schemaDoc)
	if err != nil {
		fmt.Printf("schema compile failed: %s\n", err)
		return
	}

	doc, err := p.Parse(context.Background(), []byte(`<book><title>Helium</title></book>`))
	if err != nil {
		fmt.Printf("xml parse failed: %s\n", err)
		return
	}

	if err := relaxng.NewValidator(grammar).Filename("doc.xml").Validate(context.Background(), doc); err != nil {
		fmt.Println(err)
	}
	// Output:
}

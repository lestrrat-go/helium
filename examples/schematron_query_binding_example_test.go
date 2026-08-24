package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
)

func Example_schematron_query_binding() {
	p := helium.NewParser()

	// queryBinding="xslt3" compiles the schema's expressions as XPath 3.1,
	// so the rule may call a function XPath 1.0 does not have.
	schemaDoc, err := p.Parse(context.Background(), []byte(
		`<schema xmlns="http://purl.oclc.org/dsdl/schematron" queryBinding="xslt3">
  <pattern name="isbn-check">
    <rule context="book">
      <assert test="matches(@isbn, '^[0-9]{13}$')">isbn must be 13 digits</assert>
    </rule>
  </pattern>
</schema>`))
	if err != nil {
		fmt.Printf("schema parse failed: %s\n", err)
		return
	}

	schema, err := schematron.NewCompiler().Compile(context.Background(), schemaDoc)
	if err != nil {
		fmt.Printf("schema compile failed: %s\n", err)
		return
	}
	fmt.Printf("binding: %s\n", schema.QueryBinding())

	doc, err := p.Parse(context.Background(), []byte(`<book isbn="12345"/>`))
	if err != nil {
		fmt.Printf("xml parse failed: %s\n", err)
		return
	}

	// The isbn is too short, so the assertion fires and Validate reports a
	// failure. Attach an ErrorHandler to receive the individual messages.
	if err := schematron.NewValidator(schema).Label("book.xml").Validate(context.Background(), doc); err != nil {
		fmt.Println(err)
	}

	// A binding this package does not implement is refused at compile time
	// rather than evaluated as if it were XPath 1.0. ISO/IEC 19757-3 requires
	// that refusal: only the bindings the standard defines and this package
	// implements are accepted.
	rejectedDoc, err := p.Parse(context.Background(), []byte(
		`<schema xmlns="http://purl.oclc.org/dsdl/schematron" queryBinding="xslt2">
  <pattern>
    <rule context="book">
      <assert test="@isbn">isbn is required</assert>
    </rule>
  </pattern>
</schema>`))
	if err != nil {
		fmt.Printf("schema parse failed: %s\n", err)
		return
	}

	if _, err := schematron.NewCompiler().Compile(context.Background(), rejectedDoc); err != nil {
		fmt.Println(err)
	}
	// Output:
	// binding: xpath3
	// schematron: validation failed
	// schematron: unsupported query language binding "xslt2" (supported: xslt, xslt3, xpath3)
}

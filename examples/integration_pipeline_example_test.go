package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/lestrrat-go/helium/xslt3"
)

func Example_integration_parse_validate_transform() {
	// A real-world pipeline: parse XML, validate it against an XSD schema,
	// then transform it with XSLT — all using helium packages together.

	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="order">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:simpleContent>
              <xs:extension base="xs:string">
                <xs:attribute name="qty" type="xs:integer" use="required"/>
              </xs:extension>
            </xs:simpleContent>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="order">
    <summary>
      <xsl:apply-templates select="item"/>
    </summary>
  </xsl:template>
  <xsl:template match="item">
    <line><xsl:value-of select="concat(., ' x', @qty)"/></line>
  </xsl:template>
</xsl:stylesheet>`

	const dataSrc = `<order><item qty="2">Widget</item><item qty="5">Gadget</item></order>`

	ctx := context.Background()
	p := helium.NewParser()

	// Step 1: Parse the data document.
	dataDoc, err := p.Parse(ctx, []byte(dataSrc))
	if err != nil {
		fmt.Printf("parse data failed: %s\n", err)
		return
	}

	// Step 2: Compile the XSD schema and validate.
	schemaDoc, err := p.Parse(ctx, []byte(schemaSrc))
	if err != nil {
		fmt.Printf("parse schema failed: %s\n", err)
		return
	}
	schema, err := xsd.NewCompiler().Compile(ctx, schemaDoc)
	if err != nil {
		fmt.Printf("compile schema failed: %s\n", err)
		return
	}
	if err := xsd.NewValidator(schema).Validate(ctx, dataDoc); err != nil {
		fmt.Printf("validation failed: %s\n", err)
		return
	}
	fmt.Println("valid: true")

	// Step 3: Compile the XSLT stylesheet and transform.
	stylesheetDoc, err := p.Parse(ctx, []byte(stylesheetSrc))
	if err != nil {
		fmt.Printf("parse stylesheet failed: %s\n", err)
		return
	}
	stylesheet, err := xslt3.NewCompiler().Compile(ctx, stylesheetDoc)
	if err != nil {
		fmt.Printf("compile stylesheet failed: %s\n", err)
		return
	}
	resultDoc, err := stylesheet.Transform(dataDoc).Do(ctx)
	if err != nil {
		fmt.Printf("transform failed: %s\n", err)
		return
	}

	// Step 4: Serialize the result.
	out, err := serializeExampleDocument(resultDoc)
	if err != nil {
		fmt.Printf("serialize failed: %s\n", err)
		return
	}
	fmt.Println(out)

	// Output:
	// valid: true
	// <summary><line>Widget x2</line><line>Gadget x5</line></summary>
}

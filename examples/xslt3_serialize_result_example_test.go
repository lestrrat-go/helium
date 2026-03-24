package examples_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_serialize_result() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" indent="yes" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <report>
      <item>one</item>
      <item>two</item>
    </report>
  </xsl:template>
</xsl:stylesheet>`

	ctx := context.Background()

	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	sourceDoc, err := parseExampleDocument(ctx, `<root/>`)
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// First, run the transform to get the result document.
	resultDoc, err := stylesheet.Transform(sourceDoc).Do(ctx)
	if err != nil {
		fmt.Printf("transform error: %s\n", err)
		return
	}

	// Then serialize using the stylesheet's output definition.
	// SerializeResult applies the xsl:output settings (method, indent, etc.).
	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, resultDoc, stylesheet.DefaultOutputDef())
	if err != nil {
		fmt.Printf("serialize error: %s\n", err)
		return
	}

	fmt.Println(strings.TrimSpace(buf.String()))
	// Output:
	// <report>
	//   <item>one</item>
	//   <item>two</item>
	// </report>
}

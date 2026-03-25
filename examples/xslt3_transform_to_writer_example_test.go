package examples_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_transform_to_writer() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:for-each select="items/item">
      <xsl:value-of select="concat(., '&#10;')"/>
    </xsl:for-each>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<items><item>alpha</item><item>bravo</item><item>charlie</item></items>`

	ctx := context.Background()

	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	sourceDoc, err := parseExampleDocument(ctx, sourceSrc)
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// TransformToWriter writes directly to any io.Writer.
	var buf bytes.Buffer
	err = xslt3.TransformToWriter(ctx, sourceDoc, stylesheet, &buf)
	if err != nil {
		fmt.Printf("transform error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// alpha
	// bravo
	// charlie
}

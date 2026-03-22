package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_transform() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <products>
      <xsl:apply-templates select="catalog/item"/>
    </products>
  </xsl:template>
  <xsl:template match="item">
    <product>
      <xsl:value-of select="name"/>
    </product>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<?xml version="1.0"?>
<catalog>
  <item><name>Tea</name></item>
  <item><name>Coffee</name></item>
</catalog>`

	ctx := context.Background()

	// The basic XSLT workflow is:
	// 1. parse or load the stylesheet
	// 2. compile it to *xslt3.Stylesheet
	// 3. parse the source document
	// 4. call xslt3.Transform
	stylesheetDoc, err := helium.Parse(ctx, []byte(stylesheetSrc))
	if err != nil {
		fmt.Printf("failed to parse stylesheet: %s\n", err)
		return
	}

	stylesheet, err := xslt3.CompileStylesheet(ctx, stylesheetDoc)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	sourceDoc, err := helium.Parse(ctx, []byte(sourceSrc))
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

	// Transform returns the primary result tree as a helium document. You can
	// keep working with it as a DOM or serialize it afterward.
	resultDoc, err := xslt3.Transform(ctx, sourceDoc, stylesheet)
	if err != nil {
		fmt.Printf("transform failed: %s\n", err)
		return
	}

	out, err := serializeExampleDocument(resultDoc)
	if err != nil {
		fmt.Printf("failed to serialize result: %s\n", err)
		return
	}

	fmt.Println(out)
	// Output:
	// <products><product>Tea</product><product>Coffee</product></products>
}

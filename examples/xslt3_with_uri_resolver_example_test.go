package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_with_uri_resolver() {
	const mainStylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="common.xsl"/>
  <xsl:template match="/">
    <items>
      <xsl:apply-templates select="catalog/item"/>
    </items>
  </xsl:template>
</xsl:stylesheet>`

	const includedStylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="item">
    <item>
      <xsl:value-of select="@code"/>
    </item>
  </xsl:template>
</xsl:stylesheet>`

	ctx := context.Background()

	// URI resolvers are used at stylesheet compile time for xsl:include,
	// xsl:import, and other external reads. They are a good fit when your
	// stylesheets live in memory, in an embedded filesystem, or behind a custom
	// storage layer instead of regular disk paths.
	stylesheetDoc, err := helium.NewParser().Parse(ctx, []byte(mainStylesheetSrc))
	if err != nil {
		fmt.Printf("failed to parse stylesheet: %s\n", err)
		return
	}

	// The base URI matters because relative href values such as "common.xsl"
	// are resolved against it before the resolver is called.
	stylesheet, err := xslt3.NewCompiler().
		BaseURI("/virtual/main.xsl").
		URIResolver(exampleXSLTResolver{
			"/virtual/common.xsl": includedStylesheetSrc,
		}).
		Compile(ctx, stylesheetDoc)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	sourceDoc, err := helium.NewParser().Parse(ctx, []byte(`<catalog><item code="A1"/><item code="B2"/></catalog>`))
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

	resultDoc, err := stylesheet.Transform(sourceDoc).Do(ctx)
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
	// <items><item>A1</item><item>B2</item></items>
}

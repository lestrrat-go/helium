package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_xslt3_apply_templates() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode name="summary"/>

  <xsl:template match="/" mode="summary">
    <summary>
      <xsl:apply-templates select="catalog/item" mode="summary"/>
    </summary>
  </xsl:template>

  <xsl:template match="item" mode="summary">
    <entry><xsl:value-of select="@id"/></entry>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<catalog>
  <item id="A1"/>
  <item id="B2"/>
</catalog>`

	ctx := context.Background()

	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	sourceDoc, err := helium.NewParser().Parse(ctx, []byte(sourceSrc))
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// ApplyTemplates with a named mode. Unlike Transform (which uses the
	// default mode), ApplyTemplates lets you specify which mode to use.
	resultDoc, err := stylesheet.ApplyTemplates(sourceDoc).
		Mode("summary").
		Do(ctx)
	if err != nil {
		fmt.Printf("apply-templates error: %s\n", err)
		return
	}

	out, err := serializeExampleDocument(resultDoc)
	if err != nil {
		fmt.Printf("serialize error: %s\n", err)
		return
	}

	fmt.Println(out)
	// Output:
	// <summary><entry>A1</entry><entry>B2</entry></summary>
}

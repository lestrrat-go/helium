package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xslt3_call_template() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="who" select="'stranger'"/>
  <xsl:template name="greet">
    <greeting>Hello, <xsl:value-of select="$who"/>!</greeting>
  </xsl:template>
</xsl:stylesheet>`

	ctx := context.Background()

	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	// CallTemplate invokes a named template directly, without a source document.
	// SetParameter sets global stylesheet parameters.
	resultDoc, err := stylesheet.CallTemplate("greet").
		SetParameter("who", xpath3.SingleString("Helium")).
		Do(ctx)
	if err != nil {
		fmt.Printf("call-template error: %s\n", err)
		return
	}

	out, err := serializeExampleDocument(resultDoc)
	if err != nil {
		fmt.Printf("serialize error: %s\n", err)
		return
	}

	fmt.Println(out)
	// Output:
	// <greeting>Hello, Helium!</greeting>
}

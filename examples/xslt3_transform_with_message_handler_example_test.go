package examples_test

import (
	"context"
	"fmt"
)

func Example_xslt3_transform_with_message_handler() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:message select="concat('processing ', count(catalog/item), ' items')"/>
    <summary first="{catalog/item[1]/name}"/>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<?xml version="1.0"?>
<catalog>
  <item><name>Tea</name></item>
  <item><name>Coffee</name></item>
</catalog>`

	ctx := context.Background()

	// Compile the stylesheet once, then reuse the returned *xslt3.Stylesheet
	// across multiple calls to xslt3.Transform. The receiver itself is not stored
	// on the stylesheet; it is supplied per transformation through the fluent
	// invocation API (see the .Receiver() call below).
	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	// Parse the source document that will be passed to xslt3.Transform.
	// In a real program this could come from a file, HTTP response, or another
	// in-memory pipeline stage.
	sourceDoc, err := parseExampleDocument(ctx, sourceSrc)
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

	// Attach a message receiver to the invocation to capture xsl:message
	// output during execution. This is the place to connect logging, progress
	// reporting, or application-specific diagnostics.
	//
	// Gotcha: the receiver is notified when xsl:message runs, but terminate="yes"
	// still causes Do to return an error afterward. The terminate flag
	// tells you whether the message was informational or fatal.
	resultDoc, err := stylesheet.Transform(sourceDoc).
		Receiver(&exampleMessageReceiver{}).
		Do(ctx)
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
	// message: processing 2 items (terminate=false)
	// <summary first="Tea"/>
}

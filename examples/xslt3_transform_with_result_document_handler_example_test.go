package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_transform_with_result_document_handler() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output name="plain-text" method="text" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <summary count="{count(catalog/item)}"/>
    <xsl:result-document href="reports/items.txt" format="plain-text">
      <xsl:value-of select="string-join(catalog/item/name/string(), ', ')"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<?xml version="1.0"?>
<catalog>
  <item><name>Tea</name></item>
  <item><name>Coffee</name></item>
</catalog>`

	ctx := context.Background()

	// This stylesheet writes both a primary result tree and a secondary result
	// document via xsl:result-document. That is the main use case for the result
	// document handlers shown below.
	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	sourceDoc, err := parseExampleDocument(ctx, sourceSrc)
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

	var secondaryDoc *helium.Document
	var secondaryOutputDef *xslt3.OutputDef

	// Use a result-document handler to receive each secondary output as a DOM.
	// This is useful when your application wants to decide where or how to store
	// side outputs instead of letting the stylesheet write directly to disk.
	ctx = xslt3.WithResultDocumentHandler(ctx, xslt3.ResultDocumentHandlerFunc(func(href string, doc *helium.Document) {
		if href == "reports/items.txt" {
			secondaryDoc = doc
		}
	}))

	// Pair it with WithResultDocOutputDefHandler when the secondary result may
	// use a different output method or formatting rules from the primary result.
	// Without the OutputDef, re-serializing the DOM yourself can lose details
	// such as method="text", omit-xml-declaration, indentation, or named output
	// formats selected by the stylesheet.
	ctx = xslt3.WithResultDocOutputDefHandler(ctx, func(href string, outDef *xslt3.OutputDef) {
		if href == "reports/items.txt" {
			secondaryOutputDef = outDef
		}
	})

	resultDoc, err := xslt3.Transform(ctx, sourceDoc, stylesheet)
	if err != nil {
		fmt.Printf("transform failed: %s\n", err)
		return
	}

	primaryOut, err := serializeExampleDocument(resultDoc)
	if err != nil {
		fmt.Printf("failed to serialize primary result: %s\n", err)
		return
	}
	primaryOut = strings.TrimSpace(primaryOut)

	secondaryOut, err := serializeExampleResult(secondaryDoc, secondaryOutputDef)
	if err != nil {
		fmt.Printf("failed to serialize secondary result: %s\n", err)
		return
	}

	fmt.Println(primaryOut)
	fmt.Printf("secondary (%s): %s\n", secondaryOutputDef.Method, secondaryOut)
	// Output:
	// <summary count="2"/>
	// secondary (text): Tea, Coffee
}

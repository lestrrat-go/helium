package examples_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
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

	ctx = xslt3.WithMessageHandler(ctx, xslt3.MessageHandlerFunc(func(msg string, terminate bool) {
		fmt.Printf("message: %s (terminate=%t)\n", msg, terminate)
	}))

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
	// message: processing 2 items (terminate=false)
	// <summary first="Tea"/>
}

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

	ctx = xslt3.WithResultDocumentHandler(ctx, xslt3.ResultDocumentHandlerFunc(func(href string, doc *helium.Document) {
		if href == "reports/items.txt" {
			secondaryDoc = doc
		}
	}))
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

func Example_xslt3_transform_with_raw_result_handler() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template name="numbers" as="xs:integer+">
    <xsl:sequence select="1 to 3"/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := context.Background()

	stylesheet, err := compileExampleStylesheet(ctx, stylesheetSrc)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	sourceDoc, err := parseExampleDocument(ctx, `<ignored/>`)
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

	var rawResult xpath3.Sequence
	ctx = xslt3.WithInitialTemplate(ctx, "numbers")
	ctx = xslt3.WithRawResultHandler(ctx, func(seq xpath3.Sequence) {
		rawResult = append(xpath3.Sequence(nil), seq...)
	})

	resultDoc, err := xslt3.Transform(ctx, sourceDoc, stylesheet)
	if err != nil {
		fmt.Printf("transform failed: %s\n", err)
		return
	}

	rawSummary, err := formatExampleAtomicSequence(rawResult)
	if err != nil {
		fmt.Printf("failed to describe raw result: %s\n", err)
		return
	}

	firstChild := resultDoc.FirstChild()
	if firstChild == nil {
		fmt.Println("unexpected empty result document")
		return
	}

	fmt.Printf("raw: %s\n", rawSummary)
	fmt.Printf("document text: %s\n", string(firstChild.Content()))
	// Output:
	// raw: xs:integer=1, xs:integer=2, xs:integer=3
	// document text: 1 2 3
}

func Example_xslt3_transform_with_primary_items_handler() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="json"/>
  <xsl:template match="/">
    <xsl:sequence select="map{
      'count': count(catalog/item),
      'items': array{catalog/item/name/string()}
    }"/>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<?xml version="1.0"?>
<catalog>
  <item><name>Tea</name></item>
  <item><name>Coffee</name></item>
</catalog>`

	ctx := context.Background()

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

	var primaryItems xpath3.Sequence
	ctx = xslt3.WithPrimaryItemsHandler(ctx, func(seq xpath3.Sequence) {
		primaryItems = append(xpath3.Sequence(nil), seq...)
	})

	resultDoc, err := xslt3.Transform(ctx, sourceDoc, stylesheet)
	if err != nil {
		fmt.Printf("transform failed: %s\n", err)
		return
	}

	serialized, err := serializeExampleItems(primaryItems, resultDoc, stylesheet.DefaultOutputDef())
	if err != nil {
		fmt.Printf("failed to serialize captured items: %s\n", err)
		return
	}

	fmt.Printf("captured items: %d\n", len(primaryItems))
	fmt.Printf("serialized: %s\n", serialized)
	// Output:
	// captured items: 1
	// serialized: {"count":2,"items":["Tea","Coffee"]}
}

func compileExampleStylesheet(ctx context.Context, src string) (*xslt3.Stylesheet, error) {
	doc, err := helium.Parse(ctx, []byte(src))
	if err != nil {
		return nil, err
	}
	return xslt3.CompileStylesheet(ctx, doc)
}

func parseExampleDocument(ctx context.Context, src string) (*helium.Document, error) {
	return helium.Parse(ctx, []byte(src))
}

func serializeExampleResult(doc *helium.Document, outDef *xslt3.OutputDef) (string, error) {
	var buf bytes.Buffer
	if err := xslt3.SerializeResult(&buf, doc, outDef); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func serializeExampleItems(items xpath3.Sequence, doc *helium.Document, outDef *xslt3.OutputDef) (string, error) {
	var buf bytes.Buffer
	if err := xslt3.SerializeItems(&buf, items, doc, outDef); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func formatExampleAtomicSequence(seq xpath3.Sequence) (string, error) {
	parts := make([]string, 0, len(seq))
	for _, item := range seq {
		atomic, ok := item.(xpath3.AtomicValue)
		if !ok {
			return "", fmt.Errorf("unexpected non-atomic item %T", item)
		}
		value, err := xpath3.AtomicToString(atomic)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("%s=%s", atomic.TypeName, value))
	}
	return strings.Join(parts, ", "), nil
}

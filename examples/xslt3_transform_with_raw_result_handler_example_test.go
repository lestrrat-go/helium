package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
)

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

	// This example starts from a named template that returns a typed sequence.
	// That makes it easy to see the difference between the raw XDM value and the
	// final serialized document that Transform returns.
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

	// Select a named template as the entry point, then use a raw-result handler
	// to keep the original typed XDM sequence before it is serialized into the
	// result document.
	//
	// Use this when the type matters to your application, for example if you
	// need to preserve xs:integer/xs:date/xs:decimal values instead of consuming
	// only their text form after serialization.
	//
	// Gotcha: Transform still returns a result document, so the raw handler is a
	// supplement to normal output handling, not a replacement for it.
	ctx = xslt3.WithInitialTemplate(ctx, "numbers")
	ctx = xslt3.WithRawResultHandler(ctx, func(seq xpath3.Sequence) {
		rawResult = xpath3.ItemSlice(append([]xpath3.Item(nil), seq.Materialize()...))
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

package examples_test

import (
	"context"
	"fmt"
)

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

	// This stylesheet uses method="json", which means the primary output is
	// driven by XDM items such as maps and arrays instead of an element tree.
	// That is the scenario where PrimaryItemsReceiver is most useful.
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

	// For json/adaptive output, a primary-items receiver lets callers access the
	// non-node XDM items that the serializer will turn into the final output.
	// This is useful if your program wants to inspect the map/array structure,
	// apply custom serialization, or hand the items to another layer without
	// going through text first.
	//
	// Gotcha: this callback is primarily interesting for non-XML outputs. For a
	// normal element-based XML result, the returned document is usually the more
	// natural API to consume.
	recv := &examplePrimaryItemsReceiver{}

	resultDoc, err := stylesheet.Transform(sourceDoc).
		PrimaryItemsReceiver(recv).
		Do(ctx)
	if err != nil {
		fmt.Printf("transform failed: %s\n", err)
		return
	}

	serialized, err := serializeExampleItems(recv.items, resultDoc, stylesheet.DefaultOutputDef())
	if err != nil {
		fmt.Printf("failed to serialize captured items: %s\n", err)
		return
	}

	fmt.Printf("captured items: %d\n", recv.items.Len())
	fmt.Printf("serialized: %s\n", serialized)
	// Output:
	// captured items: 1
	// serialized: {"count":2,"items":["Tea","Coffee"]}
}

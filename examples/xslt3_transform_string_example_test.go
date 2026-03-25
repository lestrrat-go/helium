package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_transform_string() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <greeting>Hello, <xsl:value-of select="person/@name"/>!</greeting>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<person name="World"/>`

	ctx := context.Background()

	stylesheetDoc, err := helium.NewParser().Parse(ctx, []byte(stylesheetSrc))
	if err != nil {
		fmt.Printf("parse stylesheet error: %s\n", err)
		return
	}

	stylesheet, err := xslt3.CompileStylesheet(ctx, stylesheetDoc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	sourceDoc, err := helium.NewParser().Parse(ctx, []byte(sourceSrc))
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// TransformString is a convenience that compiles+transforms+serializes
	// in one call, returning the result as a string.
	result, err := xslt3.TransformString(ctx, sourceDoc, stylesheet)
	if err != nil {
		fmt.Printf("transform error: %s\n", err)
		return
	}

	fmt.Println(result)
	// Output:
	// <?xml version="1.0" encoding="UTF-8"?><greeting>Hello, World!</greeting>
}

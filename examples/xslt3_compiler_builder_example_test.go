package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_compiler_builder() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="env" select="'dev'"/>
  <xsl:template match="/">
    <config env="{$env}">
      <xsl:value-of select="concat('running in ', $env)"/>
    </config>
  </xsl:template>
</xsl:stylesheet>`

	ctx := context.Background()

	doc, err := helium.Parse(ctx, []byte(stylesheetSrc))
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// The Compiler type uses a fluent builder pattern. Each method returns
	// a new copy (clone-on-write), so the original is never modified.
	// This lets you create a base compiler and derive variants from it.
	base := xslt3.NewCompiler().
		BaseURI("file:///stylesheets/config.xsl")

	stylesheet, err := base.Compile(ctx, doc)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}

	sourceDoc, err := parseExampleDocument(ctx, `<root/>`)
	if err != nil {
		fmt.Printf("parse error: %s\n", err)
		return
	}

	// Override the default "dev" value with a runtime parameter.
	resultDoc, err := stylesheet.Transform(sourceDoc).
		SetParameter("env", xpath3.SingleString("production")).
		Do(ctx)
	if err != nil {
		fmt.Printf("transform error: %s\n", err)
		return
	}

	out, err := serializeExampleDocument(resultDoc)
	if err != nil {
		fmt.Printf("serialize error: %s\n", err)
		return
	}

	fmt.Println(out)
	// Output:
	// <config env="production">running in production</config>
}

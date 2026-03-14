package examples_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

func Example_xslt3_transform() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <products>
      <xsl:apply-templates select="catalog/item"/>
    </products>
  </xsl:template>
  <xsl:template match="item">
    <product>
      <xsl:value-of select="name"/>
    </product>
  </xsl:template>
</xsl:stylesheet>`

	const sourceSrc = `<?xml version="1.0"?>
<catalog>
  <item><name>Tea</name></item>
  <item><name>Coffee</name></item>
</catalog>`

	ctx := context.Background()

	stylesheetDoc, err := helium.Parse(ctx, []byte(stylesheetSrc))
	if err != nil {
		fmt.Printf("failed to parse stylesheet: %s\n", err)
		return
	}

	stylesheet, err := xslt3.CompileStylesheet(stylesheetDoc)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	sourceDoc, err := helium.Parse(ctx, []byte(sourceSrc))
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

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
	// <products><product>Tea</product><product>Coffee</product></products>
}

func Example_xslt3_compile_file() {
	const stylesheetSrc = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="title" select="'world'"/>
  <xsl:template name="report">
    <xsl:message select="'building report'"/>
    <report>
      <xsl:value-of select="concat('Hello, ', $title)"/>
    </report>
  </xsl:template>
</xsl:stylesheet>`

	dir, err := os.MkdirTemp(".", ".tmp-xslt3-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer os.RemoveAll(dir) //nolint:errcheck

	stylesheetPath := filepath.Join(dir, "report.xsl")
	if err := os.WriteFile(stylesheetPath, []byte(stylesheetSrc), 0644); err != nil {
		fmt.Printf("failed to write stylesheet: %s\n", err)
		return
	}

	stylesheet, err := xslt3.CompileFile(stylesheetPath)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	sourceDoc, err := helium.Parse(context.Background(), []byte(`<ignored/>`))
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

	ctx := context.Background()
	ctx = xslt3.WithInitialTemplate(ctx, "report")
	ctx = xslt3.WithParameter(ctx, "title", "Helium")
	ctx = xslt3.WithMessageHandler(ctx, func(msg string, terminate bool) {
		fmt.Printf("message: %s (terminate=%t)\n", msg, terminate)
	})

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
	// message: building report (terminate=false)
	// <report>Hello, Helium</report>
}

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

	stylesheetDoc, err := helium.Parse(ctx, []byte(mainStylesheetSrc))
	if err != nil {
		fmt.Printf("failed to parse stylesheet: %s\n", err)
		return
	}

	stylesheet, err := xslt3.CompileStylesheet(
		stylesheetDoc,
		xslt3.WithBaseURI("/virtual/main.xsl"),
		xslt3.WithURIResolver(exampleXSLTResolver{
			"/virtual/common.xsl": includedStylesheetSrc,
		}),
	)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	sourceDoc, err := helium.Parse(ctx, []byte(`<catalog><item code="A1"/><item code="B2"/></catalog>`))
	if err != nil {
		fmt.Printf("failed to parse source: %s\n", err)
		return
	}

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
	// <items><item>A1</item><item>B2</item></items>
}

type exampleXSLTResolver map[string]string

func (r exampleXSLTResolver) Resolve(uri string) (io.ReadCloser, error) {
	data, ok := r[uri]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(data)), nil
}

func serializeExampleDocument(doc *helium.Document) (string, error) {
	var buf bytes.Buffer
	if err := doc.XML(&buf, helium.WithNoDecl()); err != nil {
		return "", err
	}
	return buf.String(), nil
}

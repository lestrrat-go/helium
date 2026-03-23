package examples_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
)

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

	stylesheet, err := xslt3.CompileFile(context.Background(), stylesheetPath)
	if err != nil {
		fmt.Printf("failed to compile stylesheet: %s\n", err)
		return
	}

	resultDoc, err := stylesheet.CallTemplate("report").
		SetParameter("title", xpath3.SingleString("Helium")).
		Receiver(&exampleMessageReceiver{}).
		Do(context.Background())
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

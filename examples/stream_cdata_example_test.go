package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_cdata() {
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)

	if err := w.StartElement("script"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteCDATA writes a CDATA section: <![CDATA[ ... ]]>
	// Content inside a CDATA section is not XML-escaped, so characters
	// like <, >, and & are preserved literally. This is useful for
	// embedding code snippets or other content that contains many
	// special characters.
	if err := w.WriteCDATA("if (a < b && c > d) { return true; }"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := w.EndElement(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <script><![CDATA[if (a < b && c > d) { return true; }]]></script>
}

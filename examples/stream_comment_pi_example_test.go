package examples_test

import (
	"bytes"
	"fmt"

	"github.com/lestrrat-go/helium/stream"
)

func Example_stream_comment_pi() {
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)

	if err := w.StartDocument("1.0", "", ""); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WritePI writes a processing instruction: <?target data?>
	// Processing instructions are commonly used for stylesheet
	// associations (xml-stylesheet) and other application-specific directives.
	// The PI appears in the document prolog (before the root element).
	if err := w.WritePI("xml-stylesheet", `type="text/xsl" href="style.xsl"`); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := w.StartElement("doc"); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	// WriteComment writes an XML comment: <!-- ... -->
	// The comment text should include leading/trailing spaces if desired,
	// as the writer does not add them automatically.
	if err := w.WriteComment(" this is a comment "); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	if err := w.EndDocument(); err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Print(buf.String())
	// Output:
	// <?xml version="1.0"?>
	// <?xml-stylesheet type="text/xsl" href="style.xsl"?><doc><!-- this is a comment --></doc>
}

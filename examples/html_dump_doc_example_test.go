package examples_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/html"
)

func Example_html_dump_doc() {
	// Parse HTML first, then serialize with html.DumpDoc.
	doc, err := html.Parse([]byte(`<html><body><div>Hello</div></body></html>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	var buf bytes.Buffer
	if err := html.DumpDoc(&buf, doc); err != nil {
		fmt.Printf("failed to dump: %s\n", err)
		return
	}

	out := buf.String()
	fmt.Println(strings.HasPrefix(out, "<!DOCTYPE html PUBLIC"))
	fmt.Println(strings.Contains(out, "<div>Hello</div>"))
	// Output:
	// true
	// true
}

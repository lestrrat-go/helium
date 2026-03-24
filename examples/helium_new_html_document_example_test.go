package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_new_html_document() {
	// NewHTMLDocument creates an empty document whose type is HTML rather than
	// generic XML. That is useful when you plan to build or serialize HTML
	// programmatically and want HTML-specific behavior downstream.
	doc := helium.NewHTMLDocument()
	fmt.Println(doc.Type() == helium.HTMLDocumentNode)
	// Output:
	// true
}

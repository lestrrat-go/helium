package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_new_html_document() {
	doc := helium.NewHTMLDocument()
	fmt.Println(doc.Type() == helium.HTMLDocumentNode)
	// Output:
	// true
}

package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

func Example_xpath_must_compile() {
	expr := xpath.MustCompile("count(child::*)")

	doc, err := helium.Parse([]byte(`<root><a/><b/></root>`))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	r, err := expr.Evaluate(doc.DocumentElement())
	if err != nil {
		fmt.Printf("evaluate failed: %s\n", err)
		return
	}

	fmt.Printf("%.0f\n", r.Number)
	// Output:
	// 2
}

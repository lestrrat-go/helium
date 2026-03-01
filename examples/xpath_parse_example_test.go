package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/xpath"
)

func Example_xpath_parse() {
	ast, err := xpath.Parse("count(/root/item)")
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	_, ok := ast.(xpath.FunctionCall)
	fmt.Println(ok)
	// Output:
	// true
}

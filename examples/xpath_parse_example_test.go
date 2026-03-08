package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/xpath1"
)

func Example_xpath_parse() {
	ast, err := xpath1.Parse("count(/root/item)")
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	_, ok := ast.(xpath1.FunctionCall)
	fmt.Println(ok)
	// Output:
	// true
}

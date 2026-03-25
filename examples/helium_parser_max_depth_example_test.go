package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parser_max_depth() {
	// MaxDepth is a defensive limit against excessively deep input. This is
	// useful when parsing untrusted XML or when you want to fail fast on inputs
	// that are unexpectedly nested.
	p := helium.NewParser().MaxDepth(2)

	_, err := p.Parse(context.Background(), []byte(`<root><level1><level2/></level1></root>`))
	fmt.Println(err != nil)
	fmt.Println(strings.Contains(err.Error(), "exceeded max depth"))
	// Output:
	// true
	// true
}

package examples_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parser_max_depth() {
	p := helium.NewParser()
	p.SetMaxDepth(2)

	_, err := p.Parse(context.Background(), []byte(`<root><level1><level2/></level1></root>`))
	fmt.Println(err != nil)
	fmt.Println(strings.Contains(err.Error(), "exceeded max depth"))
	// Output:
	// true
	// true
}

package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_evaluate() {
	doc, err := helium.Parse(context.Background(), []byte(`<prices><item>10</item><item>20</item><item>30</item></prices>`))
	if err != nil {
		fmt.Printf("failed to parse: %s\n", err)
		return
	}

	r, err := xpath3.Evaluate(context.Background(), doc, "string(//item[1])")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	s, ok := r.IsString()
	if !ok {
		fmt.Println("unexpected non-string result")
		return
	}
	fmt.Printf("string: %s\n", s)

	r, err = xpath3.Evaluate(context.Background(), doc, "sum(//item)")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	n, ok := r.IsNumber()
	if !ok {
		fmt.Println("unexpected non-numeric result")
		return
	}
	fmt.Printf("sum: %.0f\n", n)

	r, err = xpath3.Evaluate(context.Background(), doc, "count(//item) > 2")
	if err != nil {
		fmt.Printf("xpath error: %s\n", err)
		return
	}
	b, ok := r.IsBoolean()
	if !ok {
		fmt.Println("unexpected non-boolean result")
		return
	}
	fmt.Printf("more than 2: %t\n", b)
	// Output:
	// string: 10
	// sum: 60
	// more than 2: true
}

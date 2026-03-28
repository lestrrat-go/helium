package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_quantified() {
	// Quantified expressions test whether some or every item in a sequence
	// satisfies a condition — useful for validation-style checks.
	const src = `<scores><score>85</score><score>92</score><score>78</score></scores>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	// "some" — at least one score is above 90.
	expr, err := compiler.Compile(`some $s in //score satisfies xs:integer($s) > 90`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	b, _ := r.IsBoolean()
	fmt.Printf("some > 90: %t\n", b)

	// "every" — all scores are above 70.
	expr, err = compiler.Compile(`every $s in //score satisfies xs:integer($s) > 70`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	b, _ = r.IsBoolean()
	fmt.Printf("every > 70: %t\n", b)

	// "every" — not all scores are above 80.
	expr, err = compiler.Compile(`every $s in //score satisfies xs:integer($s) > 80`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	b, _ = r.IsBoolean()
	fmt.Printf("every > 80: %t\n", b)

	// Output:
	// some > 90: true
	// every > 70: true
	// every > 80: false
}

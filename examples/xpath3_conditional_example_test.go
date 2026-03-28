package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_conditional_and_simple_map() {
	// if/then/else selects between two branches. The simple map operator (!)
	// applies an expression to each item in a sequence — a lightweight
	// alternative to for-each.
	const src = `<catalog>
  <item type="book">Go in Action</item>
  <item type="video">XPath Crash Course</item>
  <item type="book">XML Essentials</item>
</catalog>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	// if/then/else — label each item by its type.
	expr, err := compiler.Compile(`
		string-join(
			for $i in //item return
				if ($i/@type = "book")
				then concat("[book] ", $i)
				else concat("[other] ", $i),
			"; ")
	`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err := eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	s, _ := r.IsString()
	fmt.Printf("labeled: %s\n", s)

	// Simple map operator (!) — extract text from each item.
	expr, err = compiler.Compile(`string-join(//item ! upper-case(.), ", ")`)
	if err != nil {
		fmt.Printf("compile error: %s\n", err)
		return
	}
	r, err = eval.Evaluate(context.Background(), expr, doc)
	if err != nil {
		fmt.Printf("eval error: %s\n", err)
		return
	}
	s, _ = r.IsString()
	fmt.Printf("mapped: %s\n", s)

	// Output:
	// labeled: [book] Go in Action; [other] XPath Crash Course; [book] XML Essentials
	// mapped: GO IN ACTION, XPATH CRASH COURSE, XML ESSENTIALS
}

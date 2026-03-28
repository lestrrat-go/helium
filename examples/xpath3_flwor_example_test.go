package examples_test

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func Example_xpath3_flwor() {
	// FLWOR expressions (for/let/where/order by/return) are the query engine
	// of XPath 3.1 — iterate, bind, filter, sort, and project in one expression.
	const src = `<library>
  <book price="30"><title>Go in Action</title></book>
  <book price="25"><title>XML Essentials</title></book>
  <book price="45"><title>XPath Mastery</title></book>
  <book price="20"><title>Web Basics</title></book>
</library>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	if err != nil {
		fmt.Printf("parse failed: %s\n", err)
		return
	}

	compiler := xpath3.NewCompiler()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	// Use let to bind a variable, then return a computed value.
	expr, err := compiler.Compile(`let $total := sum(//book/@price) return concat("total: $", $total)`)
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
	fmt.Printf("%s\n", s)

	// Use for to iterate with a filter and transformation.
	expr, err = compiler.Compile(`string-join(for $b in //book[xs:integer(@price) > 25] return concat($b/title, " ($", $b/@price, ")"), "; ")`)
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
	fmt.Printf("expensive: %s\n", s)

	// Simple for-return without where/order — double each number.
	expr, err = compiler.Compile(`string-join(for $x in (1, 2, 3) return string($x * 2), ",")`)
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
	fmt.Printf("doubled: %s\n", s)

	// Output:
	// total: $120
	// expensive: Go in Action ($30); XPath Mastery ($45)
	// doubled: 2,4,6
}

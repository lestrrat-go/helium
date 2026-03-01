package examples_test

import (
	"fmt"

	"github.com/lestrrat-go/helium/xpath"
)

func Example_xpath_new_lexer() {
	lexer, err := xpath.NewLexer(`/root/item`)
	if err != nil {
		fmt.Printf("lex failed: %s\n", err)
		return
	}

	first := lexer.Peek()
	fmt.Println(first.Value)
	fmt.Println(first.Value == lexer.Next().Value)
	// Output:
	// /
	// true
}

package xpath1

import "github.com/lestrrat-go/helium/internal/xpath1/lexer"

// newLexer creates a lexer and tokenizes the entire input.
func newLexer(input string) (*lexer.Lexer, error) {
	return lexer.New(input)
}

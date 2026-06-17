package xpath3

// NewLexerForTesting exposes the internal lexer for tests.
func NewLexerForTesting(input string) (*lexer, error) {
	return newLexer(input)
}

// MaxNodesForTesting overrides the sequence/node-set size limit for tests so
// limit-enforcement can be exercised without materializing millions of items.
func (e Evaluator) MaxNodesForTesting(n int) Evaluator {
	e = e.clone()
	e.cfg.maxNodes = n
	return e
}

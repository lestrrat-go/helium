package xpath3

// NewLexerForTesting exposes the internal lexer for tests.
func NewLexerForTesting(input string) (*lexer, error) {
	return newLexer(input)
}

// NewArrayBorrowingForTesting builds an ArrayItem that borrows the given member
// sequences WITHOUT cloning them (unlike NewArray, which deep-clones and would
// materialize a lazy/panic-on-materialize member at construction time). Tests
// use it to hand an un-materialized lazy member into the array lookup paths so
// only lookupItem's own bound can fire.
func NewArrayBorrowingForTesting(members []Sequence) ArrayItem {
	return ArrayItem{members: members}
}

// MaxNodesForTesting overrides the sequence/node-set size limit for tests so
// limit-enforcement can be exercised without materializing millions of items.
func (e Evaluator) MaxNodesForTesting(n int) Evaluator {
	e = e.clone()
	e.cfg.maxNodes = n
	return e
}

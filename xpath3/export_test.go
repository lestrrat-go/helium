package xpath3

import (
	"context"
	"time"
)

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

// MatchTimeoutForTesting reports the match timeout the backtracking engine
// applies to r, or zero when r compiled to Go's RE2 engine, which needs none.
func (r *Regex) MatchTimeoutForTesting() time.Duration {
	if r.inner.backtrack == nil {
		return 0
	}
	return r.inner.backtrack.MatchTimeout
}

// BacktrackingForTesting reports whether r compiled to the backtracking regexp2
// engine instead of Go's linear RE2 engine.
func (r *Regex) BacktrackingForTesting() bool {
	return r.inner.backtrack != nil
}

// CountOpsForTesting charges n operations, against limit, to an op counter that
// already stands at count. It returns the counter afterwards and the charge's
// error, so a test can start the counter next to math.MaxInt without running
// that many operations.
func CountOpsForTesting(ctx context.Context, count, n, limit int) (int, error) {
	ec := &evalContext{opCount: &count, opLimit: limit}
	err := ec.countOps(ctx, n)
	return count, err
}

// ParseSimpleIntForTesting exposes the picture-width integer parser.
var ParseSimpleIntForTesting = parseSimpleInt

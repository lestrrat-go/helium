package helium_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateNestedRepetitionBacktracking exercises content models where a
// greedy inner repetition would starve a later iteration of an outer
// repetition, which the greedy recursive-descent matcher cannot resolve on its
// own. The exact reachability fallback must accept the language members and
// still reject genuine non-members.
func TestValidateNestedRepetitionBacktracking(t *testing.T) {
	t.Parallel()

	t.Run("(lhs,(rhs,(com|wfc|vc)*)+)", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE prod [
<!ELEMENT prod (lhs,(rhs,(com|wfc|vc)*)+)>
<!ELEMENT lhs EMPTY>
<!ELEMENT rhs EMPTY>
<!ELEMENT com EMPTY>
<!ELEMENT wfc EMPTY>
<!ELEMENT vc EMPTY>
]>`
		// Single iteration of the outer plus-group.
		require.Empty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/><com/></prod>`),
			"single iteration is valid")

		// Two iterations: [rhs com] then [rhs vc]. The inner (com|wfc|vc)* of
		// the first iteration must NOT greedily swallow the second rhs.
		require.Empty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/><com/><rhs/><vc/></prod>`),
			"two iterations is valid")

		// A bare rhs with no trailing choice items (choice group is *).
		require.Empty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/></prod>`),
			"rhs with zero choice items is valid")

		// Missing the mandatory leading lhs.
		require.NotEmpty(t, parseValidating(t, dtd+`<prod><rhs/><com/></prod>`),
			"missing lhs is invalid")

		// A trailing element outside the model.
		require.NotEmpty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/><com/><bogus/></prod>`),
			"trailing undeclared element is invalid")

		// Only lhs: the outer group requires at least one rhs.
		require.NotEmpty(t, parseValidating(t, dtd+`<prod><lhs/></prod>`),
			"missing mandatory rhs is invalid")
	})

	t.Run("(a,(b,c*)+)", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE r [
<!ELEMENT r (a,(b,c*)+)>
<!ELEMENT a EMPTY>
<!ELEMENT b EMPTY>
<!ELEMENT c EMPTY>
]>`
		require.Empty(t, parseValidating(t, dtd+`<r><a/><b/></r>`), "a,b valid")
		require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><c/><b/><c/><c/></r>`),
			"a then (b,c*) twice valid")
		require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><c/><c/></r>`), "a,b,c,c valid")
		require.NotEmpty(t, parseValidating(t, dtd+`<r><a/></r>`), "missing b invalid")
		require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><c/></r>`), "c before b invalid")
	})

	t.Run("(a|b)+", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE r [
<!ELEMENT r (a|b)+>
<!ELEMENT a EMPTY>
<!ELEMENT b EMPTY>
]>`
		require.Empty(t, parseValidating(t, dtd+`<r><a/></r>`), "single a valid")
		require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><a/><a/><b/></r>`), "mixed valid")
		require.NotEmpty(t, parseValidating(t, dtd+`<r></r>`), "empty invalid (needs one)")
		require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><c/></r>`), "undeclared c invalid")
	})

	t.Run("((a,b)|c)*", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE r [
<!ELEMENT r ((a,b)|c)*>
<!ELEMENT a EMPTY>
<!ELEMENT b EMPTY>
<!ELEMENT c EMPTY>
]>`
		require.Empty(t, parseValidating(t, dtd+`<r></r>`), "empty valid (star)")
		require.Empty(t, parseValidating(t, dtd+`<r><c/></r>`), "single c valid")
		require.Empty(t, parseValidating(t, dtd+`<r><a/><b/></r>`), "a,b valid")
		require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><c/><a/><b/></r>`),
			"interleaved groups valid")
		require.NotEmpty(t, parseValidating(t, dtd+`<r><a/></r>`), "a without b invalid")
		require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><c/></r>`), "a then c (no b) invalid")
		require.NotEmpty(t, parseValidating(t, dtd+`<r><b/><a/></r>`), "b before a invalid")
	})

	// A genuine backtracking case the greedy matcher alone cannot handle even
	// with correct grouping: the optional a? greedily consumes the only a,
	// leaving the required a unmatched. The exact fallback must accept it.
	t.Run("(a?,a) backtracking", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE r [
<!ELEMENT r (a?,a)>
<!ELEMENT a EMPTY>
]>`
		require.Empty(t, parseValidating(t, dtd+`<r><a/></r>`), "single a satisfies (a?,a)")
		require.Empty(t, parseValidating(t, dtd+`<r><a/><a/></r>`), "two a satisfies (a?,a)")
		require.NotEmpty(t, parseValidating(t, dtd+`<r></r>`), "empty invalid (needs one a)")
		require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><a/><a/></r>`), "three a invalid")
	})
}

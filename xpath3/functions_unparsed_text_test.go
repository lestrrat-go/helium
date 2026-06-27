package xpath3_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// stringURIResolver returns fixed content for any URI.
type stringURIResolver struct{ content string }

func (r stringURIResolver) ResolveURI(string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(r.content)), nil
}

// fn:unparsed-text-lines over a resource with more lines than the configured
// node-set/sequence cap must reject the result via the cap rather than
// materializing the whole line sequence. Before the fix the function built an
// ItemSlice of every line with no fnMaxNodes guard, so an oversized resource
// produced an unbounded sequence.
func TestFnUnparsedTextLines_HonorsMaxNodes(t *testing.T) {
	const lines = 1000
	const limit = 20

	var b strings.Builder
	for range lines {
		b.WriteString("line\n")
	}
	resolver := stringURIResolver{content: b.String()}

	compiled, err := xpath3.NewCompiler().Compile("unparsed-text-lines('http://example.com/lines.txt')")
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		URIResolver(resolver).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
}

// fn:unparsed-text-lines over a many-line resource with a small OpLimit must
// reject via the op limit AND must not first allocate a []string proportional to
// the resource's full line count. The line production is bounded by the
// EFFECTIVE budget (min of maxNodes and the remaining op budget), so an
// OpLimit far below the 10M default maxNodes stops splitting after ~OpLimit
// lines. Before the fix the splitter was handed maxNodes (10M), so a 1M-line
// resource built a 1M-entry slice before the op charge rejected it — the
// allocation count below catches that regression (it grows with the line count
// under the bug, stays tiny under the fix).
func TestFnUnparsedTextLines_BoundsAllocationByOpLimit(t *testing.T) {
	const lines = 1_000_000
	const opLimit = 1000

	var b strings.Builder
	for range lines {
		b.WriteString("x\n")
	}
	resolver := stringURIResolver{content: b.String()}

	compiled, err := xpath3.NewCompiler().Compile("unparsed-text-lines('http://example.com/lines.txt')")
	require.NoError(t, err)

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		URIResolver(resolver).
		OpLimit(opLimit)

	var evalErr error
	allocs := testing.AllocsPerRun(1, func() {
		_, evalErr = eval.Evaluate(t.Context(), compiled, nil)
	})
	require.ErrorIs(t, evalErr, xpath3.ErrOpLimit)
	// Under the fix the splitter produces ~opLimit lines, so allocations stay a
	// few orders of magnitude below the 1M line count. A threshold well under the
	// line count cleanly separates the bounded fix from the proportional bug.
	require.Less(t, allocs, float64(100_000),
		"line splitting must be bounded by the active op limit, not the resource line count")
}

// A resource whose line count is within the cap must succeed and return every
// line, confirming the bound does not reject legitimate results.
func TestFnUnparsedTextLines_WithinMaxNodes(t *testing.T) {
	const lines = 5
	const limit = 20

	var b strings.Builder
	for range lines {
		b.WriteString("line\n")
	}
	resolver := stringURIResolver{content: b.String()}

	compiled, err := xpath3.NewCompiler().Compile("unparsed-text-lines('http://example.com/lines.txt')")
	require.NoError(t, err)

	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		URIResolver(resolver).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)
	require.Equal(t, lines, res.Sequence().Len())
}

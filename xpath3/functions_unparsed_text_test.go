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

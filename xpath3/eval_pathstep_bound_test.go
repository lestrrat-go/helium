package xpath3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// A path step E1/E2 whose right side yields atomic (non-node) items must enforce
// the configured node-set/sequence limit on the accumulated result, just as the
// node-producing branch does. Each per-node range (1 to 5) stays under the cap,
// so the only thing that can overflow is evalPathStepExpr's accumulation of the
// atomic results across all base nodes. Before the fix this branch appended
// without any bound and returned an oversized sequence; now it must trip
// ErrNodeSetLimit.
func TestEvalPathStepExpr_AtomicResultHonorsMaxNodes(t *testing.T) {
	// Base node count and per-node range size are each kept under the cap so
	// neither the base node-set evaluation nor any single range trips the limit;
	// only the path-step's accumulation across base nodes (5*5 = 25 > 20) can.
	const aCount = 5
	const limit = 20

	var b strings.Builder
	b.WriteString("<root>")
	for range aCount {
		b.WriteString("<a/>")
	}
	b.WriteString("</root>")

	doc := mustParseXML(t, b.String())
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile("(a)/(1 to 5)")
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, root)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
}

// A path step whose accumulated atomic result stays within the cap must succeed
// and return the full sequence, confirming the bound does not reject legitimate
// results.
func TestEvalPathStepExpr_AtomicResultWithinMaxNodes(t *testing.T) {
	const aCount = 3
	const limit = 20 // aCount*2 = 6 <= limit

	var b strings.Builder
	b.WriteString("<root>")
	for range aCount {
		b.WriteString("<a/>")
	}
	b.WriteString("</root>")

	doc := mustParseXML(t, b.String())
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile("(a)/(1 to 2)")
	require.NoError(t, err)

	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, root)
	require.NoError(t, err)
	require.Equal(t, aCount*2, res.Sequence().Len())
}

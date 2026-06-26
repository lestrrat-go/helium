package xpath3_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// buildWideElementDoc returns "<root><c/>...<c/></root>" with width child
// elements. fn:idref walks every element of the document, so a wide tree forces
// the walk to visit many nodes and lets the per-node cancellation / op-limit
// check fire before the whole tree is scanned.
func buildWideElementDoc(width int) string {
	var b strings.Builder
	b.WriteString("<root>")
	for range width {
		b.WriteString("<c/>")
	}
	b.WriteString("</root>")
	return b.String()
}

// fn:idref walks the entire document collecting IDREF matches. That walk must
// honor context cancellation: a context cancelled after evaluation begins must
// abort the walk promptly with context.Canceled rather than scanning every node
// and silently swallowing the error returned by the tree walk.
func TestFnIDRef_ContextCancelledWideDoc(t *testing.T) {
	const width = 50000
	const cancelAfter = 50

	doc := mustParseXML(t, buildWideElementDoc(width))
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile(`idref("x")`)
	require.NoError(t, err)

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, root)
	require.ErrorIs(t, err, context.Canceled)
	// The walk consults ctx.Err() once per visited node, so the cancel-after
	// context reaches its trip count mid-walk and the walk bails on the first
	// cancelled observation. Without the in-walk check the walk would scan all
	// width nodes, swallow the error, and Evaluate would return success.
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"idref walk must bail on the first cancelled Err() observation")
}

// fn:idref's document walk must be charged against the established op-limit so a
// large document cannot run the walk unbounded. The walk previously neither
// counted ops nor propagated the error returned by the tree walk.
func TestFnIDRef_OpLimitWideDoc(t *testing.T) {
	const width = 50000
	const opLimit = 200

	doc := mustParseXML(t, buildWideElementDoc(width))
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile(`idref("x")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		OpLimit(opLimit).
		Evaluate(t.Context(), compiled, root)
	require.ErrorIs(t, err, xpath3.ErrOpLimit)
}

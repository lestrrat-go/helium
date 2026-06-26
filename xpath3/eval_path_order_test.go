package xpath3_test

import (
	"context"
	"slices"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// customReverseFn returns its node-sequence argument in reverse order. It stands
// in for a user function that shadows fn:reverse when the "fn" prefix is rebound.
type customReverseFn struct{}

func (customReverseFn) MinArity() int { return 1 }
func (customReverseFn) MaxArity() int { return 1 }

func (customReverseFn) Call(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	nodes, _ := xpath3.NodesFrom(args[0])
	out := make(xpath3.ItemSlice, 0, len(nodes))
	for _, n := range slices.Backward(nodes) {
		out = append(out, xpath3.NodeItem{Node: n})
	}
	return out, nil
}

// TestFilterPreservesOrderRuntimeBinding guards against treating a path filter
// as the order-controlling built-in fn:reverse/fn:sort based on its lexical
// spelling. The order-preservation decision must consult the runtime function
// binding: when the "fn" prefix is rebound (or a user function overrides the
// built-in), normal document-order deduplication must apply to the path result.
func TestFilterPreservesOrderRuntimeBinding(t *testing.T) {
	doc := mustParseXML(t, `<root><item id="a"/><item id="b"/><item id="c"/></root>`)

	ids := func(seq xpath3.Sequence) []string {
		t.Helper()
		nodes, ok := xpath3.NodesFrom(seq)
		require.True(t, ok)
		out := make([]string, 0, len(nodes))
		for _, n := range nodes {
			el, ok := n.(*helium.Element)
			require.True(t, ok)
			v, _ := el.GetAttribute("id")
			out = append(out, v)
		}
		return out
	}

	t.Run("genuine fn:reverse preserves caller order", func(t *testing.T) {
		seq := evalExpr(t, doc, `fn:reverse(//item)/self::item`)
		require.Equal(t, []string{"c", "b", "a"}, ids(seq))
	})

	t.Run("rebound fn prefix with user reverse uses document order", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(map[string]string{"fn": "urn:custom"}).
			Functions(nil, map[xpath3.QualifiedName]xpath3.Function{
				{URI: "urn:custom", Name: "reverse"}: customReverseFn{},
			})
		seq := evalExprWithEval(t, eval, doc, `fn:reverse(//item)/self::item`)
		require.Equal(t, []string{"a", "b", "c"}, ids(seq))
	})
}

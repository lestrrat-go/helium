package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// A simple-map expression (E1 ! E2) whose right side yields a large lazy
// sequence must enforce the configured node-set/sequence limit BEFORE
// materializing the whole right-hand sequence. The right side is bound to a lazy
// Range that counts how many items it actually produces; the bound (20) is far
// below the range length (1000). The lazy accumulation must stop as soon as the
// running total would exceed the cap, so only a handful of items are ever
// produced. Before the fix the operator materialized the entire right-hand
// sequence first and only then checked the aggregate cap, producing all 1000
// items before erroring.
func TestEvalSimpleMapExpr_LazyRightHonorsMaxNodesBeforeMaterialize(t *testing.T) {
	const rangeLen = 1000
	const limit = 20

	var produced int
	big := sequence.NewRange[xpath3.Item](rangeLen, func(int) xpath3.Item {
		produced++
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "x"}
	})

	compiled, err := xpath3.NewCompiler().Compile("1 ! $big")
	require.NoError(t, err)

	// EvalBorrowing keeps $big lazy (DefaultEvaluatorOptions would clone and thus
	// materialize the sequence up front, defeating the laziness probe).
	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(map[string]xpath3.Sequence{"big": big}).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
	// The cap must trip after producing only a bounded prefix, never the whole
	// 1000-item range.
	require.Less(t, produced, rangeLen, "right-hand sequence was fully materialized before the cap check")
	require.LessOrEqual(t, produced, limit+1)
}

// A simple-map whose accumulated result stays within the cap must succeed and
// return the full sequence, confirming the bound does not reject legitimate
// results.
func TestEvalSimpleMapExpr_WithinMaxNodes(t *testing.T) {
	const limit = 20

	compiled, err := xpath3.NewCompiler().Compile("(1, 2, 3) ! (1 to 2)")
	require.NoError(t, err)

	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)
	require.Equal(t, 6, res.Sequence().Len())
}

// A FLWOR return clause has the same materialize-before-cap hazard as
// simple-map: `for $i in 1 return $big` with a borrowed lazy $big must enforce
// the node-set/sequence limit item-by-item BEFORE the return sub-sequence is
// materialized. The right side is bound to a lazy Range that counts how many
// items it actually produces; the bound (20) is far below the range length
// (1000). Before the fix the return clause called seqMaterialize first, so all
// 1000 items were produced before the aggregate cap rejected the result.
func TestEvalFLWOR_LazyReturnHonorsMaxNodesBeforeMaterialize(t *testing.T) {
	const rangeLen = 1000
	const limit = 20

	var produced int
	big := sequence.NewRange[xpath3.Item](rangeLen, func(int) xpath3.Item {
		produced++
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "x"}
	})

	compiled, err := xpath3.NewCompiler().Compile("for $i in 1 return $big")
	require.NoError(t, err)

	// EvalBorrowing keeps $big lazy (DefaultEvaluatorOptions would clone and thus
	// materialize the sequence up front, defeating the laziness probe).
	_, err = xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(map[string]xpath3.Sequence{"big": big}).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.ErrorIs(t, err, xpath3.ErrNodeSetLimit)
	require.Less(t, produced, rangeLen, "return sub-sequence was fully materialized before the cap check")
	require.LessOrEqual(t, produced, limit+1)
}

// A FLWOR whose accumulated result stays within the cap must succeed and return
// the full sequence, confirming the bound does not reject legitimate results.
func TestEvalFLWOR_WithinMaxNodes(t *testing.T) {
	const limit = 20

	compiled, err := xpath3.NewCompiler().Compile("for $i in (1, 2, 3) return (1 to 2)")
	require.NoError(t, err)

	res, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		MaxNodesForTesting(limit).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)
	require.Equal(t, 6, res.Sequence().Len())
}

// Labels unionLabel gives to text, comment and namespaced attribute nodes.
const (
	unionText    = "text()"
	unionComment = "comment()"
	unionPC      = "@p:c"
)

// unionOrderSrc gives the element e attributes, a namespace declaration and
// mixed children, next to a sibling f with its own attribute and child.
const unionOrderSrc = `<r xmlns:p="urn:p"><e a="1" b="2" p:c="3">t1<x id="x1"/><!--c--><x id="x2"/></e><f g="4"><y id="y"/></f></r>`

// unionLabel names n for TestEvalUnionExpr: an attribute by its name, an
// element by its id (or name), a namespace node by its prefix, and other
// nodes by their kind.
func unionLabel(n helium.Node) string {
	switch n.Type() {
	case helium.AttributeNode:
		return "@" + n.Name()
	case helium.NamespaceNode:
		return "ns:" + n.Name()
	case helium.TextNode:
		return unionText
	case helium.CommentNode:
		return unionComment
	}
	e, ok := n.(*helium.Element)
	if !ok {
		return n.Name()
	}
	if v, ok := e.GetAttribute("id"); ok {
		return v
	}
	return e.Name()
}

// evalUnionLabels evaluates expr from ctxNode and labels every node of the
// result, in result order.
func evalUnionLabels(t *testing.T, eval xpath3.Evaluator, expr string, ctxNode helium.Node) []string {
	t.Helper()
	r, err := eval.Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(expr), ctxNode)
	require.NoError(t, err, expr)
	nodes, err := r.Nodes()
	require.NoError(t, err, expr)
	labels := make([]string, len(nodes))
	for i, n := range nodes {
		labels[i] = unionLabel(n)
	}
	return labels
}

// TestEvalUnionExpr pins the order and deduplication of union, intersect and
// except results for operands that are already in document order and for
// operands that need sorting or deduplication.
func TestEvalUnionExpr(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(unionOrderSrc))
	require.NoError(t, err)
	e := doc.DocumentElement().FirstChild()
	require.Equal(t, "e", e.Name())
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	all := []string{"@a", "@b", unionPC, unionText, "x1", unionComment, "x2"}
	cases := []struct {
		expr string
		want []string
	}{
		{expr: "@*|node()", want: all},
		{expr: "node()|@*", want: all},
		{expr: "@*|node()|@*", want: all},
		{expr: "@b|@a", want: []string{"@a", "@b"}},
		{expr: "@a|@b|@a", want: []string{"@a", "@b"}},
		{expr: "@a|node()", want: []string{"@a", unionText, "x1", unionComment, "x2"}},
		{expr: "*[2]|*[1]", want: []string{"x1", "x2"}},
		{expr: "*[1]|*[1]", want: []string{"x1"}},
		{expr: "*|text()", want: []string{unionText, "x1", "x2"}},
		{expr: "(*, *)|@a", want: []string{"@a", "x1", "x2"}},
		{expr: "(*[2], *[1])|@b", want: []string{"@b", "x1", "x2"}},
		{expr: "@*|(node(), node())", want: all},
		{expr: "reverse(node())|reverse(@*)", want: all},
		{expr: "namespace::p|@a", want: []string{"ns:p", "@a"}},
		{expr: "@a|namespace::p", want: []string{"ns:p", "@a"}},
		{expr: "namespace::p|namespace::p", want: []string{"ns:p"}},
		{expr: "@*|../f/@*", want: []string{"@a", "@b", unionPC, "@g"}},
		{expr: "../f/@*|@*", want: []string{"@a", "@b", unionPC, "@g"}},
		{expr: "@a|../f/node()", want: []string{"@a", "y"}},
		{expr: "../f/node()|@a", want: []string{"@a", "y"}},
		{expr: "..|@a", want: []string{"r", "@a"}},
		{expr: "@a|self::node()", want: []string{"e", "@a"}},
		{expr: "x/@*|@*", want: []string{"@a", "@b", unionPC, "@id", "@id"}},
		{expr: "descendant::node()|@a", want: []string{"@a", unionText, "x1", unionComment, "x2"}},
		{expr: "(@*|node()) intersect node()", want: []string{unionText, "x1", unionComment, "x2"}},
		{expr: "(@*|node()) except @*", want: []string{unionText, "x1", unionComment, "x2"}},
		{expr: "(@*, node()) intersect (node(), @*)", want: all},
		{expr: "reverse(node()) except text()", want: []string{"x1", unionComment, "x2"}},
	}
	for _, c := range cases {
		require.Equal(t, c.want, evalUnionLabels(t, eval, c.expr, e), c.expr)
	}

	t.Run("operands from two documents", func(t *testing.T) {
		other, err := helium.NewParser().Parse(t.Context(), []byte(`<o k="v"><z id="z1"/></o>`))
		require.NoError(t, err)
		vars := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(map[string]xpath3.Sequence{
			"other": xpath3.ItemSlice{xpath3.NodeItem{Node: other.DocumentElement()}},
		})
		// Each evaluation uses a fresh cache, and the document that cache
		// meets first sorts first.
		require.Equal(t, []string{"@a", "@b", unionPC, "@k"}, evalUnionLabels(t, vars, "$other/@*|@*", e))
		require.Equal(t, []string{"@k", "z1", "@a"}, evalUnionLabels(t, vars, "$other/(@*|node())|@a", e))
	})

	t.Run("a shared cache keeps the document order of a fast union", func(t *testing.T) {
		doc1, err := helium.NewParser().Parse(t.Context(), []byte(`<a><b id="x1"/><b id="x2"/></a>`))
		require.NoError(t, err)
		doc2, err := helium.NewParser().Parse(t.Context(), []byte(`<a k="v"><b id="y1"/></a>`))
		require.NoError(t, err)
		shared := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(map[string]xpath3.Sequence{
			"other": xpath3.ItemSlice{xpath3.NodeItem{Node: doc1.DocumentElement()}},
		}).DocOrderCache(xpath3.NewDocOrderCache())

		// Each step of this union selects one node, so the union is the first
		// expression to meet doc2 and must register it ahead of doc1.
		require.Equal(t, []string{"@k", "y1"}, evalUnionLabels(t, shared, "@*|node()", doc2.DocumentElement()))
		require.Equal(t, []string{"y1", "x1", "x2"}, evalUnionLabels(t, shared, "$other/b | /a/b", doc2))
	})

	t.Run("node limit", func(t *testing.T) {
		limited := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).MaxNodesForTesting(3)
		for _, expr := range []string{"@*|node()", "node()|@*", "@*|*|@*", "(@a, @b)|(@c, *)"} {
			_, err := limited.Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(expr), e)
			require.ErrorIs(t, err, xpath3.ErrNodeSetLimit, expr)
		}
		require.Equal(t, []string{"@a", "@b", unionPC}, evalUnionLabels(t, limited, "@*|@a", e))
		require.Equal(t, []string{"@a", "x1", "x2"}, evalUnionLabels(t, limited, "@a|*", e))
	})
}

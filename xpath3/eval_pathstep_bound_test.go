package xpath3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
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

// pathStepMixCase is one path expression evaluated from the document node of
// pathStepMixDoc. A case wants either an error code or a rendered result.
type pathStepMixCase struct {
	name string
	expr string
	code string
	want string
}

const pathStepMixDoc = `<r><a id="1"/><a id="2"/></r>`

// XPath 3.1 §3.3.1.1: the last step of E1/E2 may return nodes or non-nodes but
// not both (XPTY0018), whether the mix comes from one evaluation of E2 or from
// evaluations for different context items. Every step other than the last must
// return nodes only (XPTY0019). An empty step result counts as either kind.
var pathStepMixCases = []pathStepMixCase{
	{name: "last step mixed within one context item", expr: "/r/(1, .)", code: "XPTY0018"},
	{name: "last step mixed within each context item", expr: "/r/a/(., 'x')", code: "XPTY0018"},
	{name: "last step attribute and string", expr: "/r/a/(@id, string(@id))", code: "XPTY0018"},
	{name: "last step node then atomic across items", expr: "/r/a/(if (@id = '1') then . else 1)", code: "XPTY0018"},
	{name: "last step atomic then node across items", expr: "/r/a/(if (@id = '1') then 1 else .)", code: "XPTY0018"},
	{name: "intermediate step mixed within one context item", expr: "/r/(1, .)/a", code: "XPTY0018"},
	{name: "intermediate step mixed across items", expr: "/r/a/(if (@id = '1') then . else 1)/@id", code: "XPTY0018"},
	{name: "intermediate step mixed first operand", expr: "(1, /r)/a", code: "XPTY0019"},
	{name: "intermediate step mixed first operand before non-axis step", expr: "(/r, 1)/(a)", code: "XPTY0019"},
	{name: "intermediate step atomic before axis step", expr: "/r/a/string(@id)/a", code: "XPTY0019"},
	{name: "intermediate step atomic before non-axis step", expr: "/r/a/(1)/(.)", code: "XPTY0019"},
	{name: "last step all atomic", expr: "/r/a/(1, 2)", want: "xs:integer(1) xs:integer(2) xs:integer(1) xs:integer(2)"},
	{name: "last step all strings", expr: "/r/a/string(@id)", want: "xs:string(1) xs:string(2)"},
	{name: "last step empty then atomic", expr: "/r/a/(if (@id = '1') then () else 'x')", want: "xs:string(x)"},
	{name: "last step empty then node", expr: "/r/a/(if (@id = '1') then () else .)", want: "a#2"},
	{name: "last step all nodes sorted", expr: "/r/a/(., ..)", want: "r a#1 a#2"},
	{name: "last step all nodes deduplicated", expr: "/r/a/(.., /r)", want: "r"},
}

// pathStepCompilers compiles an expression from its string, and from its
// parsed AST, so both lowering routes into the evaluator are covered.
var pathStepCompilers = []struct {
	name    string
	compile func(*testing.T, string) *xpath3.Expression
}{
	{name: "Compile", compile: compilePathStepString},
	{name: "CompileExpr", compile: compilePathStepAST},
}

func compilePathStepString(t *testing.T, expr string) *xpath3.Expression {
	t.Helper()
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	return compiled
}

func compilePathStepAST(t *testing.T, expr string) *xpath3.Expression {
	t.Helper()
	compiled, err := xpath3.NewCompiler().CompileExpr(mustParseExpr(t, expr))
	require.NoError(t, err)
	return compiled
}

// renderPathStepItems renders an element as its name plus its id attribute,
// and an atomic value with its type.
func renderPathStepItems(t *testing.T, seq xpath3.Sequence) string {
	t.Helper()
	var parts []string
	for item := range seq.Items() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			e, ok := v.Node.(*helium.Element)
			require.True(t, ok, "unexpected node %s", v.Node.Name())
			id, ok := e.GetAttribute("id")
			if !ok {
				parts = append(parts, e.Name())
				continue
			}
			parts = append(parts, e.Name()+"#"+id)
		case xpath3.AtomicValue:
			parts = append(parts, v.String())
		default:
			require.Failf(t, "unexpected item", "%T", item)
		}
	}
	return strings.Join(parts, " ")
}

func TestEvalPathStepExpr_NodeAtomicMix(t *testing.T) {
	doc := mustParseXML(t, pathStepMixDoc)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	for _, c := range pathStepCompilers {
		t.Run(c.name, func(t *testing.T) {
			for _, tc := range pathStepMixCases {
				t.Run(tc.name, func(t *testing.T) {
					res, err := eval.Evaluate(t.Context(), c.compile(t, tc.expr), doc)
					if tc.code != "" {
						requireErrorCode(t, err, tc.code)
						return
					}
					require.NoError(t, err)
					require.Equal(t, tc.want, renderPathStepItems(t, res.Sequence()))
				})
			}
		})
	}
}

// A non-node result of a step other than the last still matches the
// ErrPathNotNodeSet sentinel.
func TestEvalPathStepExpr_IntermediateNonNodeSentinel(t *testing.T) {
	doc := mustParseXML(t, pathStepMixDoc)
	for _, expr := range []string{"(1, /r)/a", "/r/a/(1)/(.)"} {
		_, err := evaluate(t.Context(), doc, expr)
		require.ErrorIs(t, err, xpath3.ErrPathNotNodeSet, expr)
	}
}

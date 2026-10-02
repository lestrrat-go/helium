package xpath3_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// descendantUnfused spells out `//` with an extra self::node() step. The
// self step keeps every node, so the expression selects what the `//` form
// selects, but descendant-or-self::node() is no longer followed by a child
// or attribute step and the path is evaluated one step at a time. The self
// step charges one operation per node of the descendant-or-self result.
const descendantUnfused = "/descendant-or-self::node()/self::node()/"

// descendantFusionExprs are expressions using `//` in front of a child or
// attribute step. Each {D} stands for `//`; the reference expression replaces
// it with descendantUnfused.
var descendantFusionExprs = []string{
	"{D}b", "{D}*", "{D}node()", "{D}text()", "{D}comment()", "{D}processing-instruction()",
	"{D}@id", "{D}@*", "{D}attribute()", "{D}element()", "{D}c", "{D}nosuch", "{D}p:x", "{D}*:x",
	"{D}b[1]", "{D}b[last()]", "{D}*[2]", "{D}*[position() = 2]", "{D}node()[last()]",
	"{D}b[@id]", "{D}b[@id = 'b1']", "{D}*[b][1]", "{D}*[.{D}c]", "{D}b[c][last()]",
	"{D}@*[1]", "{D}@*[last()]", "{D}@id[. = 'b1']", "{D}*[@id][2]",
	"{D}b[xs:integer(substring(@id, 2)) > 1]",
	".{D}b", ".{D}@id", ".{D}*[1]", "/a{D}b", "/a{D}@*", "/*/*{D}node()", "..{D}b",
	"{D}b/c", "{D}b{D}c", "{D}*{D}@id", "{D}b/..", "{D}b{D}c[1]",
	// Predicates that only select, which the walk applies in one pass.
	"{D}b[@id][1]", "{D}*[@id = 'b1'][1]", "{D}*[2][@id]", "{D}*[@p:k]", "{D}*[@id = 'nosuch']",
	"{D}@*[2]", "{D}node()[1][@id]", "{D}*[@*][last()]",
	// `//` from several context nodes: siblings, text and attribute nodes
	// (which take the step-by-step evaluation), and nested nodes.
	"/a/b{D}c", "/*/*{D}node()", "/*/*{D}@*", "/*/*{D}*[1]", "/*/*{D}*[last()]", "/*/node(){D}node()",
	"/*/*{D}b[@id]", "{D}text(){D}node()", "{D}@id{D}node()", "/*/*{D}*{D}c", "count(/*/*{D}*)",
	"(/*/*{D}*)[2]", "/*/*{D}b | $other/*{D}b", "{D}*[@id]{D}b[1]",
	"{D}a{D}b", "{D}*{D}*", "{D}*{D}node()", "{D}node(){D}node()", "{D}node(){D}@*", "{D}a{D}b[1]",
	"{D}a{D}b[last()]", "{D}*{D}*[@id]", "{D}*{D}b[@id = 'b2']", "{D}a{D}b[.{D}c]", "{D}*{D}*[2][@id]",
	"/descendant::*{D}c", "/descendant::*{D}@id", "/descendant::node(){D}node()", "{D}b/descendant::*{D}c",
	"count({D}*{D}*)", "({D}*{D}b)[2]", "/*/descendant-or-self::*{D}b", "{D}a[b]{D}*[1]",
	"$other{D}b", "$nodes{D}b", "$nodes{D}@id", "$ents{D}node()",
	"count({D}*)", "count({D}node())", "count({D}@*)", "count({D}b[@id])", "count({D}nosuch)",
	"exists({D}b)", "exists({D}nosuch)", "empty({D}b)", "empty({D}nosuch)",
	"boolean({D}c)", "not({D}c)", "head({D}b)", "head({D}nosuch)", "fn:count({D}b)",
	"string-join({D}b/@id, ',')", "sum({D}*/count(@*))",
	"({D}b)[1]", "({D}b)[2]", "({D}b)[2.0]", "({D}b)[0]", "({D}b)[1.5]", "({D}b)[last()]",
	"({D}b)[1][1]", "({D}b)['x']", "({D}*)[@id][2]",
	// The registration order of documents in the order cache decides the
	// order of a union of nodes from two documents, so each `//` must
	// register its document where the two-step evaluation does.
	"{D}b | $other{D}b", "$other{D}b | {D}b", "({D}nosuch, $other{D}b | {D}b)",
	"(count({D}nosuch), $other{D}c | {D}c)", "({D}nosuch[1], $other{D}c | {D}c)",
	"({D}*[@id = 'b4'], $other{D}b | {D}b)", "{D}b[exists($other{D}c)] | $other{D}c",
	"{D}b[$other{D}c] | $other{D}c",
}

// TestDescendantStepFusion checks that a `//` path selects exactly what the
// same path evaluated one step at a time selects, in the same order, with
// the same error, over documents with namespaces, mixed content, entity
// references and DTDs, from every kind of context node.
func TestDescendantStepFusion(t *testing.T) {
	t.Parallel()
	for _, d := range stepOrderDocs {
		f := newStepOrderFixture(t, d)
		f.eval = f.eval.Namespaces(map[string]string{"p": "urn:p"})
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr, c.pick)
			if ctxNode == nil {
				continue
			}
			for _, tmpl := range descendantFusionExprs {
				fused := strings.ReplaceAll(tmpl, "{D}", "//")
				unfused := strings.ReplaceAll(tmpl, "{D}", descendantUnfused)
				want := f.describeResult(t, ctxNode, unfused, false)
				got := f.describeResult(t, ctxNode, fused, false)
				require.Equal(t, want, got, "%s|%s|%s", d.name, c.name, fused)
			}
		}
	}
}

// TestDescendantStepFusionRandom repeats TestDescendantStepFusion over
// generated documents.
func TestDescendantStepFusionRandom(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(3, 4)) //nolint:gosec // deterministic test input
	for range 20 {
		f := newStepOrderFixture(t, stepOrderDoc{name: "random", src: randomStepOrderDoc(rng)})
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr, c.pick)
			if ctxNode == nil {
				continue
			}
			for _, tmpl := range descendantFusionExprs {
				fused := strings.ReplaceAll(tmpl, "{D}", "//")
				unfused := strings.ReplaceAll(tmpl, "{D}", descendantUnfused)
				require.Equal(t, f.describeResult(t, ctxNode, unfused, false), f.describeResult(t, ctxNode, fused, false), fused)
			}
		}
	}
}

// TestDescendantStepFusionAnnotated repeats TestDescendantStepFusion from
// the document node with a type annotation on every attribute, so a
// predicate comparing an attribute with a string ([@id = 'b1']) evaluates
// the comparison instead of selecting by the attribute's text.
func TestDescendantStepFusionAnnotated(t *testing.T) {
	t.Parallel()
	for _, d := range stepOrderDocs {
		f := newStepOrderFixture(t, d)
		r, err := evalWith(t, f.eval, f.doc, "//@*")
		require.NoError(t, err)
		attrs, err := r.Nodes()
		require.NoError(t, err)
		annotations := make(map[helium.Node]string, len(attrs))
		for _, a := range attrs {
			annotations[a] = "xs:string"
		}
		f.eval = f.eval.Namespaces(map[string]string{"p": "urn:p"}).TypeAnnotations(annotations)
		for _, tmpl := range descendantFusionExprs {
			fused := strings.ReplaceAll(tmpl, "{D}", "//")
			unfused := strings.ReplaceAll(tmpl, "{D}", descendantUnfused)
			require.Equal(t, f.describeResult(t, f.doc, unfused, false), f.describeResult(t, f.doc, fused, false), "%s|%s", d.name, fused)
		}
	}
}

// TestDescendantNestedContextsAllocate checks that `//` from nested context
// nodes walks the outermost subtrees once: over 1,000 nested elements, the
// fused form allocates at least 500 times less than the step-by-step form,
// which traverses the subtree of every context node into a slice of its own.
// It measures allocations, so it does not run in parallel.
func TestDescendantNestedContextsAllocate(t *testing.T) {
	var b strings.Builder
	b.WriteString("<r>")
	for range 500 {
		b.WriteString("<a><a><b/></a><b/></a>")
	}
	b.WriteString("</r>")
	doc := parseStepOrderDoc(t, b.String(), false)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	for _, tmpl := range []string{"count({D}a{D}b)", "count({D}a{D}b[1])", "count({D}a{D}*[b])", "count({D}a{D}@*)"} {
		fused := xpath3.NewCompiler().MustCompile(strings.ReplaceAll(tmpl, "{D}", "//"))
		unfused := xpath3.NewCompiler().MustCompile(strings.ReplaceAll(tmpl, "{D}", descendantUnfused))
		fusedAllocs := testing.AllocsPerRun(5, func() {
			_, err := eval.Evaluate(t.Context(), fused, doc)
			require.NoError(t, err)
		})
		unfusedAllocs := testing.AllocsPerRun(5, func() {
			_, err := eval.Evaluate(t.Context(), unfused, doc)
			require.NoError(t, err)
		})
		require.Less(t, fusedAllocs+500, unfusedAllocs, tmpl)
	}
}

// The paths descendantLimitCases run their `//` from.
const (
	dosRoot          = "/"
	dosGrandchildren = "/*/*"
	dosElements      = "/descendant::*"
)

// descendantLimitCases are `//` expressions whose operation charges and
// node-set limits TestDescendantStepFusionLimits compares with the
// one-step-at-a-time evaluation. The first {D} is the one under test; dos is
// the path it runs from. A path whose value only decides whether it selects
// a node stops at its first node instead (pathExists), so the cases count the
// nodes of such a path rather than test them; eval_path_exists_test.go
// covers the early stop.
var descendantLimitCases = []struct {
	tmpl string
	dos  string
}{
	{"{D}b", dosRoot}, {"{D}*", dosRoot}, {"{D}node()", dosRoot}, {"{D}@id", dosRoot}, {"{D}@*", dosRoot},
	{"{D}b[1]", dosRoot}, {"{D}b[last()]", dosRoot}, {"{D}*[@id]", dosRoot}, {"{D}*[@id][last()]", dosRoot},
	{"{D}@*[last()]", dosRoot}, {"{D}*[count(.{D}c) > 0]", dosRoot}, {"count({D}*)", dosRoot}, {"head({D}b)", dosRoot},
	{"({D}b)[1]", dosRoot}, {"({D}*)[last()]", dosRoot}, {"{D}b{D}c", dosRoot},
	// Predicates that only select.
	{"{D}b[@id][1]", dosRoot}, {"{D}*[@id = 'b2']", dosRoot}, {"{D}*[2][@id]", dosRoot}, {"{D}@*[1]", dosRoot},
	// Several context nodes.
	{"/a/b{D}c", "/a/b"}, {"/*/node(){D}node()", "/*/node()"}, {"/*{D}b{D}c", "/*"},
	{"/*/*{D}node()", dosGrandchildren}, {"/*/*{D}@*", dosGrandchildren}, {"/*/*{D}*[1]", dosGrandchildren},
	{"/*/*{D}*[last()]", dosGrandchildren}, {"/*/*{D}*[@id]", dosGrandchildren},
	{"count(/*/*{D}*)", dosGrandchildren}, {"/*/*{D}b{D}c", dosGrandchildren},
	// Nested context nodes.
	{"/descendant::*{D}b", dosElements}, {"/descendant::*{D}@id", dosElements},
	{"/descendant::*{D}*[1]", dosElements}, {"/descendant::*{D}*[last()]", dosElements},
	{"/descendant::*{D}*[@id]", dosElements}, {"/descendant::*{D}*[count(.{D}c) > 0]", dosElements},
	{"count(/descendant::*{D}node())", dosElements}, {"/descendant::node(){D}node()", "/descendant::node()"},
}

// TestDescendantStepFusionLimits checks that a `//` path charges the same
// number of operations as the path evaluated one step at a time, and fails
// on the same node-set limits. The reference expression's extra self step
// charges one operation per node of the descendant-or-self result, so its
// smallest passing operation limit is that many operations higher.
func TestDescendantStepFusionLimits(t *testing.T) {
	t.Parallel()
	for _, d := range stepOrderDocs {
		doc := parseStepOrderDoc(t, d.src, d.subst)
		if d.build != nil {
			d.build(t, doc)
		}
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
		for _, tc := range descendantLimitCases {
			fused := strings.ReplaceAll(tc.tmpl, "{D}", "//")
			// Spell out only the `//` under test, so the extra self step
			// runs exactly once.
			unfused := strings.ReplaceAll(strings.Replace(tc.tmpl, "{D}", descendantUnfused, 1), "{D}", "//")
			name := d.name + "|" + fused
			dosExpr := "count(" + tc.dos + "/descendant-or-self::node())"
			if tc.dos == dosRoot {
				dosExpr = "count(/descendant-or-self::node())"
			}
			dosSize := evalCount(t, eval, doc, dosExpr)

			fusedOps := smallestOpLimit(t, doc, fused)
			unfusedOps := smallestOpLimit(t, doc, unfused)
			require.Equal(t, unfusedOps-dosSize, fusedOps, name)

			// One operation short of the smallest passing limit fails.
			_, err := evalWith(t, eval.OpLimit(fusedOps-1), doc, fused)
			require.ErrorIs(t, err, xpath3.ErrOpLimit, name)

			// Both forms fail when the descendant-or-self result exceeds the
			// limit, and agree on the limits around it.
			for _, limit := range []int{dosSize - 1, dosSize, dosSize + 1} {
				if limit < 1 {
					continue
				}
				limited := eval.MaxNodesForTesting(limit)
				_, fusedErr := evalWith(t, limited, doc, fused)
				_, unfusedErr := evalWith(t, limited, doc, unfused)
				require.Equal(t, errors.Is(unfusedErr, xpath3.ErrNodeSetLimit), errors.Is(fusedErr, xpath3.ErrNodeSetLimit), "%s maxNodes=%d", name, limit)
				if limit < dosSize {
					require.ErrorIs(t, fusedErr, xpath3.ErrNodeSetLimit, "%s maxNodes=%d", name, limit)
				}
			}
		}
	}
}

// TestDescendantStepFusionCancel checks that a cancelled context stops a
// `//` path.
func TestDescendantStepFusionCancel(t *testing.T) {
	t.Parallel()
	doc := parseStepOrderDoc(t, stepOrderDocs[0].src, false)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, expr := range []string{"//b", "//b[1]", "//b[@id]", "//@id", "count(//*)", "//*//b", "//*//b[1]", "//*//b[c]"} {
		compiled := xpath3.NewCompiler().MustCompile(expr)
		_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, doc)
		require.ErrorIs(t, err, context.Canceled, expr)
	}
}

// countPlusHundred is a user function that shadows fn:count.
type countPlusHundred struct{}

func (countPlusHundred) MinArity() int { return 1 }
func (countPlusHundred) MaxArity() int { return 1 }
func (countPlusHundred) Call(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return xpath3.SingleInteger(int64(args[0].Len()) + 100), nil
}

// TestDescendantStepFusionUserFunction checks that a user function that
// shadows a built-in still receives the node items of a `//` argument.
func TestDescendantStepFusionUserFunction(t *testing.T) {
	t.Parallel()
	doc := parseStepOrderDoc(t, stepOrderDocs[0].src, false)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Functions(map[string]xpath3.Function{"count": countPlusHundred{}}, nil)
	require.Equal(t, 104, evalCount(t, eval, doc, "count(//b)"))
	require.Equal(t, 4, evalCount(t, eval, doc, "fn:count(//b)"))
}

func evalWith(t *testing.T, eval xpath3.Evaluator, node helium.Node, expr string) (*xpath3.Result, error) {
	t.Helper()
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err, expr)
	return eval.Evaluate(t.Context(), compiled, node)
}

func evalCount(t *testing.T, eval xpath3.Evaluator, node helium.Node, expr string) int {
	t.Helper()
	r, err := evalWith(t, eval, node, expr)
	require.NoError(t, err, expr)
	seq := r.Sequence()
	require.Equal(t, 1, seq.Len(), expr)
	av, ok := seq.Get(0).(xpath3.AtomicValue)
	require.True(t, ok, expr)
	return int(av.ToFloat64())
}

// smallestOpLimit returns the smallest operation limit under which expr
// evaluates without ErrOpLimit.
func smallestOpLimit(t *testing.T, node helium.Node, expr string) int {
	t.Helper()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	_, err := evalWith(t, eval, node, expr)
	require.NoError(t, err, expr)
	lo, hi := 1, 1
	for {
		_, err := evalWith(t, eval.OpLimit(hi), node, expr)
		if err == nil {
			break
		}
		require.ErrorIs(t, err, xpath3.ErrOpLimit, expr)
		lo = hi + 1
		hi *= 2
	}
	for lo < hi {
		mid := lo + (hi-lo)/2
		_, err := evalWith(t, eval.OpLimit(mid), node, expr)
		if err == nil {
			hi = mid
			continue
		}
		require.ErrorIs(t, err, xpath3.ErrOpLimit, expr)
		lo = mid + 1
	}
	return hi
}

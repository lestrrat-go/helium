package xpath3_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// existsPaths are location paths and path expressions in every shape the
// early stop handles: `//` walks with and without predicates, from one and
// from several (nested) context nodes, steps run depth first after them,
// steps run one at a time before them, reverse and other axes, and path
// expressions from one and from several nodes.
var existsPaths = []string{
	"//b", "//c", "//nosuch", "//*", "//node()", "//@id", "//@*", "//text()", "//comment()",
	"//processing-instruction()", "//p:x", "//*:x",
	"//b[1]", "//b[last()]", "//b[@id]", "//b[@id = 'b3']", "//*[c]", "//b[c][last()]", "//*[@id][2]",
	"//b[number(substring(@id, 2)) > 2]", "//b[number(substring(@id, 2)) > 100]", "//*[.//c]",
	"//*[not(.//c)]", "//@*[last()]", "//b[c]/c[1]", "//b[.//c][1]/@id",
	"//b/c", "//b/nosuch", "//b//c", "//a//b[1]", "//*//@id", "//b/..", "//b/@id", "//a/b/c", "//a/b//c",
	"//b/following-sibling::*", "//b/self::b[c]", "//a//b//c", "/a//b//c",
	".//c", ".//nosuch", "./b", "b", "*", "node()", "attribute::id", "@*", "..", "ancestor::*", "ancestor::*[last()]",
	"following::*", "preceding::node()[1]", "descendant::c", exprDescendantOrSelfNode, "self::node()",
	"/a/b", "/a/b/c", "/*/*/*", "/a/*[2]/node()", "/descendant::b/c", "/descendant::*//c", "/a/b[2]//c[last()]",
	"$other//b", "$other/b", "$other/nosuch", "$nodes//c", "$nodes/b", "$ents/node()", "$ents//node()",
	"(//b)[1]//c", "(//b)[last()]/c", "reverse(//b)/c", "reverse(//b)/@*",
}

// tmplExists tests a path {X} with fn:exists.
const tmplExists = "exists({X})"

// randomDocName names a generated document.
const randomDocName = "random"

// existsNamespaces binds the prefix existsPaths use.
var existsNamespaces = map[string]string{"p": "urn:p"}

// existsTemplates place a path {X} where only whether it selects a node
// matters, and the last ones evaluate a union across two documents after it,
// whose order depends on where the path registers its document in the order
// cache.
var existsTemplates = []string{
	tmplExists, "not({X})", "if ({X}) then 1 else 2", "//*[{X}]",
	"(exists({X}), //b | $other//b)", "(exists({X}), $other//c | //c)",
}

// existsTemplatesDoc are further places tried from the document node only.
var existsTemplatesDoc = []string{
	"empty({X})", "boolean({X})", "fn:exists({X})", "Q{http://www.w3.org/2005/xpath-functions}not({X})",
	"{X} and true()", "false() or {X}", "some $i in (1, 2) satisfies {X}", "every $i in (1, 2) satisfies {X}",
	"(//*)[{X}]", "//b[exists({X})]", "(not({X}), //b | $other//b)",
}

// existsForms returns an early-stop template with {X} replaced by path, and
// its reference, where `if (true()) then X else ()` hands the whole result of
// X on as a sequence, so X is evaluated in full.
func existsForms(tmpl, path string) (string, string) {
	fast := strings.ReplaceAll(tmpl, "{X}", path)
	ref := strings.ReplaceAll(tmpl, "{X}", "(if (true()) then "+path+" else ())")
	return fast, ref
}

// TestPathExists checks that every path of existsPaths, in every place of
// existsTemplates, evaluates exactly like its reference, over documents with
// namespaces, mixed content, entity references and DTDs, from every kind of
// context node.
func TestPathExists(t *testing.T) {
	t.Parallel()
	for _, d := range stepOrderDocs {
		f := newStepOrderFixture(t, d)
		f.eval = f.eval.Namespaces(existsNamespaces)
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr, c.pick)
			if ctxNode == nil {
				continue
			}
			templates := existsTemplates
			if c.name == "doc" {
				templates = append(templates[:len(templates):len(templates)], existsTemplatesDoc...)
			}
			for _, tmpl := range templates {
				for _, path := range existsPaths {
					fast, ref := existsForms(tmpl, path)
					want := f.describeResult(t, ctxNode, ref, false)
					got := f.describeResult(t, ctxNode, fast, false)
					require.Equal(t, want, got, "%s|%s|%s", d.name, c.name, fast)
				}
			}
		}
	}
}

// TestPathExistsRandom repeats TestPathExists over generated documents.
func TestPathExistsRandom(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(7, 8)) //nolint:gosec // deterministic test input
	for range 20 {
		f := newStepOrderFixture(t, stepOrderDoc{name: randomDocName, src: randomStepOrderDoc(rng)})
		f.eval = f.eval.Namespaces(existsNamespaces)
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr, c.pick)
			if ctxNode == nil {
				continue
			}
			for _, tmpl := range existsTemplates {
				for _, path := range existsPaths {
					fast, ref := existsForms(tmpl, path)
					require.Equal(t, f.describeResult(t, ctxNode, ref, false), f.describeResult(t, ctxNode, fast, false), fast)
				}
			}
		}
	}
}

// randomExistsPath returns a random relative or absolute location path of
// one to four steps, mixing `/` and `//`, axes that run depth first and axes
// that do not, and predicates on position, attributes and nested paths.
func randomExistsPath(rng *rand.Rand) string {
	starts := []string{"", "/", "//", ".//", "$other//", "$nodes/"}
	steps := []string{
		"b", "c", "a", "*", "node()", "text()", "./@id", "@*", "..", ".", "descendant::b", "ancestor::*",
		"following-sibling::*", "preceding::node()", "self::b", "parent::*", exprDescendantOrSelfNode,
	}
	preds := []string{
		"[1]", "[last()]", "[2]", "[@id]", "[@id = 'b3']", "[c]", "[.//c]", "[not(b)]", "[position() > 1]",
		"[count(*) > 0]", "[$other//c]", "[exists(.//b)]",
	}
	var b strings.Builder
	start := starts[rng.IntN(len(starts))]
	b.WriteString(start)
	for i := range 1 + rng.IntN(4) {
		if i > 0 {
			if rng.IntN(2) == 0 {
				b.WriteString("//")
			} else {
				b.WriteString("/")
			}
		}
		b.WriteString(steps[rng.IntN(len(steps))])
		for range rng.IntN(3) {
			b.WriteString(preds[rng.IntN(len(preds))])
		}
	}
	return b.String()
}

// TestPathExistsRandomPaths repeats TestPathExists with generated paths.
func TestPathExistsRandomPaths(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(9, 10)) //nolint:gosec // deterministic test input
	paths := make([]string, 400)
	for i := range paths {
		paths[i] = randomExistsPath(rng)
	}
	docs := slices.Clone(stepOrderDocs)
	for range 5 {
		docs = append(docs, stepOrderDoc{name: randomDocName, src: randomStepOrderDoc(rng)})
	}
	for _, d := range docs {
		f := newStepOrderFixture(t, d)
		for _, c := range stepOrderContexts[:2] {
			ctxNode := f.contextNode(t, c.expr, c.pick)
			if ctxNode == nil {
				continue
			}
			for _, path := range paths {
				for _, tmpl := range []string{tmplExists, "(not({X}), $other//c | //c)", "//*[{X}]"} {
					fast, ref := existsForms(tmpl, path)
					want := f.describeResult(t, ctxNode, ref, false)
					got := f.describeResult(t, ctxNode, fast, false)
					require.Equal(t, want, got, "%s|%s|%s", d.name, c.name, fast)
				}
			}
		}
	}
}

// TestPathExistsLimits checks that a path that stops at its first node does
// no more work than its full evaluation, count() of the same path: it needs
// no more operations and only fails on a node-set limit the full evaluation
// fails on too. A path that selects nothing does exactly the work of its full
// evaluation, so it needs the same operations and fails on the same node-set
// limits.
func TestPathExistsLimits(t *testing.T) {
	t.Parallel()
	for _, d := range stepOrderDocs {
		f := newStepOrderFixture(t, d)
		f.eval = f.eval.Namespaces(existsNamespaces)
		for _, path := range existsPaths {
			fast := "exists(" + path + ")"
			full := "count(" + path + ")"
			name := d.name + "|" + fast
			r, err := evalWith(t, f.eval, f.doc, full)
			if err != nil {
				continue
			}
			count, ok := r.Sequence().Get(0).(xpath3.AtomicValue)
			require.True(t, ok, name)
			if count.ToFloat64() > 0 {
				requireNoMoreWork(t, f.eval, f.doc, fast, full, name)
				continue
			}
			require.Equal(t, smallestOpLimitWith(t, f.eval, f.doc, full), smallestOpLimitWith(t, f.eval, f.doc, fast), name)
			for _, limit := range []int{1, 2, 3, 5, 8, 13} {
				limited := f.eval.MaxNodesForTesting(limit)
				_, fastErr := evalWith(t, limited, f.doc, fast)
				_, fullErr := evalWith(t, limited, f.doc, full)
				require.Equal(t, fmt.Sprint(fullErr), fmt.Sprint(fastErr), "%s maxNodes=%d", name, limit)
			}
		}
	}
}

// earlyStopDoc returns a document of n <b> elements, each holding a <c>.
func earlyStopDoc(t *testing.T, n int) *helium.Document {
	t.Helper()
	var b strings.Builder
	b.WriteString("<r>")
	for i := range n {
		fmt.Fprintf(&b, `<b id="%d"><c/></b>`, i)
	}
	b.WriteString("</r>")
	return mustParseXML(t, b.String())
}

// earlyStopCases place a path that selects a node near the start of
// earlyStopDoc where only whether it selects a node matters.
var earlyStopCases = []struct{ tmpl, path string }{
	{tmplExists, "//b"}, {tmplExists, "//c"}, {"boolean({X})", "//@id"}, {"not({X})", "//b/c"},
	{"empty({X})", "/r/b/c"}, {"if ({X}) then 1 else 2", "//b/c[1]"}, {"/r[{X}]", ".//c"},
	{tmplExists, "//b[@id = '3']/c"}, {tmplExists, "/r/b[@id]/c[1]"}, {tmplExists, "//b/@id"},
}

// TestPathExistsStopsEarly checks that a path whose value only decides
// whether it selects a node stops at its first node. A step with predicates
// that depend on position, or on more than the node itself, still builds the
// whole candidate list of each context node, so these paths only use steps
// that take one node at a time once the list is long. Over 2,000 elements,
// each path stays within an operation limit and a node-set limit of 100 that
// its full evaluation exceeds: limits the full evaluation would hit no longer
// fire when the answer is known before reaching them. That is the intended
// trade; XPath 3.1 §2.3.4 lets an implementation skip work that cannot change
// the result.
func TestPathExistsStopsEarly(t *testing.T) {
	t.Parallel()
	doc := earlyStopDoc(t, 2000)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	for _, tc := range earlyStopCases {
		fast, ref := existsForms(tc.tmpl, tc.path)
		want, err := evalWith(t, eval, doc, ref)
		require.NoError(t, err, ref)
		got, err := evalWith(t, eval.OpLimit(100), doc, fast)
		require.NoError(t, err, fast)
		requireSameItems(t, want.Sequence(), got.Sequence(), fast)
		got, err = evalWith(t, eval.MaxNodesForTesting(100), doc, fast)
		require.NoError(t, err, fast)
		requireSameItems(t, want.Sequence(), got.Sequence(), fast)

		_, err = evalWith(t, eval.OpLimit(100), doc, ref)
		require.ErrorIs(t, err, xpath3.ErrOpLimit, ref)
		_, err = evalWith(t, eval.MaxNodesForTesting(100), doc, ref)
		require.ErrorIs(t, err, xpath3.ErrNodeSetLimit, ref)
	}
}

// TestPathExistsSkippedErrors checks the dynamic errors that the early stop
// no longer raises: a predicate that fails only for nodes after the first
// selected one is never evaluated for them, which XPath 3.1 §2.3.4 permits.
// The full evaluation still raises the error.
func TestPathExistsSkippedErrors(t *testing.T) {
	t.Parallel()
	doc := earlyStopDoc(t, 10)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	for _, tc := range []struct{ tmpl, path, code string }{
		{tmplExists, "//b[if (@id = '0') then true() else error()]", "FOER0000"},
		{tmplExists, "//b[xs:integer(@id) idiv (xs:integer(@id) - 1) = 0]", "FOAR0001"},
		{"//r[{X}]", "b/c[if (../@id = '0') then true() else error()]", "FOER0000"},
	} {
		fast, ref := existsForms(tc.tmpl, tc.path)
		r, err := evalWith(t, eval, doc, fast)
		require.NoError(t, err, fast)
		require.NotZero(t, sequenceLen(r.Sequence()), fast)

		_, err = evalWith(t, eval, doc, ref)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr, ref)
		require.Equal(t, tc.code, xerr.Code, ref)
	}
}

// TestPathExistsCancel checks that a cancelled context stops a path that
// stops at its first node, before it starts and in the middle of a walk.
func TestPathExistsCancel(t *testing.T) {
	t.Parallel()
	doc := earlyStopDoc(t, 200)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	exprs := []string{"exists(//nosuch)", "exists(//b/nosuch)", "exists(/r/b/nosuch)", "//r[.//nosuch]", "exists(//b//nosuch)"}
	for _, expr := range exprs {
		_, err := eval.Evaluate(ctx, xpath3.NewCompiler().MustCompile(expr), doc)
		require.ErrorIs(t, err, context.Canceled, expr)
	}
	for _, expr := range exprs {
		compiled := xpath3.NewCompiler().MustCompile(expr)
		counter := heliumtest.NewPollContext(t.Context(), 0, nil)
		_, err := eval.Evaluate(counter, compiled, doc)
		require.NoError(t, err, expr)
		polls := counter.Polls()
		require.Greater(t, polls, 400, expr)

		pc := heliumtest.NewPollContext(t.Context(), polls/2, context.Canceled)
		_, err = eval.Evaluate(pc, compiled, doc)
		require.ErrorIs(t, err, context.Canceled, expr)
		require.LessOrEqual(t, pc.PollsAfterExpiry(), 1, expr)
	}
}

package xpath1_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// Compile folds a predicate-free descendant-or-self::node() step (the `//`
// abbreviation) into the step after it. These tests check the fold through
// the public API only:
//
//   - The differential test compares each expression with an unfolded
//     reference: the same expression with every `//` spelled as
//     `/descendant-or-self::node()[true()]/`. The `[true()]` predicate keeps
//     every node, so the reference selects what the unfolded path selects,
//     and Compile never folds a step that carries a predicate.
//   - The shape tests read the fold from the op count. A fold removes one
//     walk of the subtree, so a folded expression passes at a lower
//     Evaluator.OpLimit than its unfolded reference `(/descendant-or-self::node())/…`,
//     which runs the same two walks through a path expression Compile does
//     not fold. An unfolded expression passes at exactly the same limit.

// foldDocs are the fixed documents of the differential test: nested
// same-name elements, attributes and namespaces, comments and PIs, and an
// entity reference whose internal subset holds a comment and a PI before the
// entity declaration.
var foldDocs = []struct {
	name     string
	xml      string
	contexts int
}{
	{
		name:     "nested",
		contexts: 5,
		xml: `<!DOCTYPE a [<!ATTLIST a id ID #IMPLIED>]>` +
			`<a id="r"><b><a id="x"><b/></a></b><c/><b xml:lang="en">t</b></a>`,
	},
	{
		name:     "attributes and namespaces",
		contexts: 4,
		xml: `<root xmlns:p="urn:p"><a x="1" p:y="2"><b p:z="3">t</b><!--c1--></a>` +
			`<p:a xmlns:q="urn:q"><b/><?pi d?></p:a><!--c2--><?pi e?><x/><x><y/><x><y/></x></x></root>`,
	},
	{
		name:     "entity reference",
		contexts: 7,
		xml: `<!DOCTYPE a [<!--subset comment--><?subset pi?><!ENTITY e "<b>ent</b><c/><b/>">` +
			`<!ATTLIST a id ID #IMPLIED>]><a id="x"><b>&e;</b><a id="y"><b/>&e;</a></a>`,
	},
}

// foldExprs mixes the shapes Compile folds with the shapes it must keep.
var foldExprs = []string{
	// Paths the fold does not touch.
	"//a", "//a/b", "//a/..", "//a/parent::*", "//a/@*", "//a/namespace::*",
	"//*/self::b", "//b/following-sibling::*", "//b/preceding-sibling::*",
	"ancestor::*", "ancestor-or-self::node()", "preceding::*", "following::*",
	"//a | //b", "(//a)[2]/b", "//a[1]", "//a[b][2]", "id('x')/..", "/a/b/c",
	"/a/b/@id", "/a/*/namespace::*", "//*[@id]/ancestor::*", "$nodes/b", "//.", ".//x",

	// Paths the fold applies to.
	"//x", "//x//y", "//a//b", "//self::b", "//descendant::b",
	"//descendant-or-self::b", ".//b", ".//.", "//.//b", "$nodes//b", "id('x')//b",
	"//node()", "//text()", "//comment()", "//processing-instruction()",
	"//processing-instruction('pi')", ".//node()", ".//comment()",
	".//processing-instruction()", ".//self::node()", "//@*", "//p:a", "//*[@p:y]",
	"//b[@x='1']", "//b[@p:z='3']", "//a[b]", "//a[not(b)]", "//b[normalize-space()]",
	"//b[string(.)='t']", "//a[.//b]", "//a[b and not(c)]", "//a[@id or b]",
	"//b[id('x')]", "//b[lang('en')]", "//b[starts-with(name(), 'b')]",
	"//a[count(b) > 0]", "//*[local-name()='a']", "count(//*[local-name()='b'])",
	"descendant-or-self::node()/child::b", "//b[boolean(1)]", "//a[@id][b]",
	"//a[.//b[@x]]", "//x[y | z]", "//b[concat(name(), 'q') = 'bq']",

	// Paths the fold must keep, because the predicate may observe position.
	"//a[last()]", "//a[position()=1]", "//b[position() mod 2 = 0]", "//b[last() - 1]",
	"//b[2]", "//a[$nodes]", "//a[count(b)]", "//a[-1]", "//b[string-length()]",
	"//a[b[1]]", "//..", "descendant-or-self::node()[1]/b", "//a[1]/b",
	"//b[f()]", "//b[ext:g()]", "//a[(b)[1]]", "//x[number(.)]", "//x[1 + 0]",
}

// unfoldedReference spells every `//` of src, and every explicit
// descendant-or-self::node() step, with a `[true()]` predicate, which keeps
// the step's nodes and stops Compile from folding it.
func unfoldedReference(src string) string {
	out := strings.ReplaceAll(src, "descendant-or-self::node()/", "descendant-or-self::node()[true()]/")
	return strings.ReplaceAll(out, "//", "/descendant-or-self::node()[true()]/")
}

// positionIsOne is a custom function that reads the context position, so
// folding a predicate that calls it would change the selected nodes.
func positionIsOne(ctx context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
	fctx := xpath1.GetFunctionContext(ctx)
	return &xpath1.Result{Type: xpath1.BooleanResult, Bool: fctx.Position() == 1}, nil
}

// foldEvaluator binds everything the expressions above reference: the p and
// ext prefixes, two position-reading custom functions, a string variable
// $one, and $nodes, a node-set spanning two documents.
func foldEvaluator(nodes []helium.Node) xpath1.Evaluator {
	return xpath1.NewEvaluator().
		Namespaces(map[string]string{"p": "urn:p", "ext": "urn:ext"}).
		Function("f", xpath1.FunctionFunc(positionIsOne)).
		FunctionNS("urn:ext", "g", xpath1.FunctionFunc(positionIsOne)).
		Variables(map[string]any{"nodes": nodes, "one": "1"})
}

// firstNodeOf returns the first node expr selects from doc, or nil.
func firstNodeOf(t *testing.T, doc *helium.Document, expr string) helium.Node {
	t.Helper()
	r, err := xpath1.Evaluate(t.Context(), doc, expr)
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	if len(r.NodeSet) == 0 {
		return nil
	}
	return r.NodeSet[0]
}

// firstEntityRef returns the first entity-reference node under n in document
// order, or nil. XPath never selects entity references, so the tree is walked
// directly.
func firstEntityRef(n helium.Node) helium.Node {
	for c := range helium.Children(n) {
		if c.Type() == helium.EntityRefNode {
			return c
		}
		if found := firstEntityRef(c); found != nil {
			return found
		}
	}
	return nil
}

type foldContext struct {
	name string
	node helium.Node
}

// foldContexts returns the document, a mid-document element, an attribute, a
// namespace node, an entity reference, the entity it refers to, and the DTD
// of doc, skipping any kind the document does not have. XPath never selects
// the last three, but a caller can pass any node as the context node, and
// xmldsig1 evaluates its XPath filter on every node of a subtree, entity
// references and entities included.
func foldContexts(t *testing.T, doc *helium.Document) []foldContext {
	t.Helper()
	candidates := []foldContext{
		{name: "document", node: doc},
		{name: "element", node: firstNodeOf(t, doc, "/*/*[1]")},
		{name: "attribute", node: firstNodeOf(t, doc, "//@*")},
		{name: "namespace", node: firstNodeOf(t, doc, "/*/*[last()]/namespace::*[last()]")},
		{name: "entity reference", node: firstEntityRef(doc)},
	}
	if ref := firstEntityRef(doc); ref != nil {
		candidates = append(candidates, foldContext{name: "entity", node: ref.FirstChild()})
	}
	if dtd := doc.IntSubset(); dtd != nil {
		candidates = append(candidates, foldContext{name: "dtd", node: dtd})
	}
	var out []foldContext
	for _, c := range candidates {
		if c.node != nil {
			out = append(out, c)
		}
	}
	return out
}

// sameNode reports whether a and b are the same node. Namespace nodes are
// built fresh on each evaluation, so they compare by owner, prefix and URI.
func sameNode(a, b helium.Node) bool {
	wa, okA := a.(*helium.NamespaceNodeWrapper)
	wb, okB := b.(*helium.NamespaceNodeWrapper)
	if okA != okB {
		return false
	}
	if !okA {
		return a == b
	}
	return wa.Parent() == wb.Parent() && wa.Name() == wb.Name() && string(wa.Content()) == string(wb.Content())
}

// requireSameResult requires two evaluation outcomes to match exactly: the
// same error text, or the same result type and value, with node-sets equal
// element by element in the same order.
func requireSameResult(t *testing.T, label string, want *xpath1.Result, wantErr error, got *xpath1.Result, gotErr error) {
	t.Helper()
	if wantErr != nil || gotErr != nil {
		require.Error(t, wantErr, label)
		require.Error(t, gotErr, label)
		require.Equal(t, wantErr.Error(), gotErr.Error(), label)
		return
	}
	require.Equal(t, want.Type, got.Type, label)
	switch want.Type {
	case xpath1.NodeSetResult:
		require.Len(t, got.NodeSet, len(want.NodeSet), label)
		for i := range want.NodeSet {
			require.True(t, sameNode(want.NodeSet[i], got.NodeSet[i]), "%s: node %d differs", label, i)
		}
	case xpath1.NumberResult:
		if math.IsNaN(want.Number) {
			require.True(t, math.IsNaN(got.Number), label)
			return
		}
		require.Equal(t, want.Number, got.Number, label)
	case xpath1.BooleanResult:
		require.Equal(t, want.Bool, got.Bool, label)
	case xpath1.StringResult:
		require.Equal(t, want.String, got.String, label)
	}
}

// requireFoldMatchesReference evaluates every fold expression and its
// unfolded reference from each context node. Both must give the same result
// in the same order.
func requireFoldMatchesReference(t *testing.T, eval xpath1.Evaluator, docName string, contexts []foldContext) {
	t.Helper()
	for _, src := range foldExprs {
		compiled, err := xpath1.Compile(src)
		require.NoError(t, err, src)
		reference, err := xpath1.Compile(unfoldedReference(src))
		require.NoError(t, err, src)
		for _, c := range contexts {
			label := fmt.Sprintf("%s / %s / %s", docName, c.name, src)
			want, wantErr := eval.Evaluate(t.Context(), reference, c.node)
			got, gotErr := eval.Evaluate(t.Context(), compiled, c.node)
			requireSameResult(t, label, want, wantErr, got, gotErr)
		}
	}
}

func parseFoldDoc(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
	require.NoError(t, err)
	return doc
}

func TestCompileFoldMatchesUnfolded(t *testing.T) {
	docs := make([]*helium.Document, len(foldDocs))
	for i, d := range foldDocs {
		docs[i] = parseFoldDoc(t, d.xml)
	}
	// $nodes spans two documents.
	nodes := []helium.Node{docs[0].DocumentElement(), docs[1].DocumentElement()}
	eval := foldEvaluator(nodes)

	for i, d := range foldDocs {
		t.Run(d.name, func(t *testing.T) {
			contexts := foldContexts(t, docs[i])
			require.Len(t, contexts, d.contexts)
			requireFoldMatchesReference(t, eval, d.name, contexts)
		})
	}

	t.Run("random documents", func(t *testing.T) {
		rng := rand.New(rand.NewPCG(1, 2))
		for i := range 40 {
			doc := parseFoldDoc(t, randomFoldDoc(rng))
			requireFoldMatchesReference(t, eval, fmt.Sprintf("random %d", i), foldContexts(t, doc))
		}
	})
}

// randomFoldDoc builds a small random document from the element names the
// fold expressions use, with attributes, text, comments and PIs mixed in.
func randomFoldDoc(rng *rand.Rand) string {
	var sb strings.Builder
	sb.WriteString(`<root xmlns:p="urn:p">`)
	writeRandomFoldChildren(rng, &sb, 0)
	sb.WriteString(`</root>`)
	return sb.String()
}

var randomFoldElementNames = []string{"a", "b", "c", "x", "y", "p:a"}

func writeRandomFoldChildren(rng *rand.Rand, sb *strings.Builder, depth int) {
	if depth > 4 {
		return
	}
	for range rng.IntN(4) {
		switch rng.IntN(6) {
		case 0:
			sb.WriteString("t")
		case 1:
			sb.WriteString("<!--c-->")
		case 2:
			sb.WriteString("<?pi d?>")
		default:
			name := randomFoldElementNames[rng.IntN(len(randomFoldElementNames))]
			sb.WriteString("<" + name)
			if rng.IntN(2) == 0 {
				fmt.Fprintf(sb, ` x="%d"`, rng.IntN(2))
			}
			if rng.IntN(3) == 0 {
				sb.WriteString(` id="x"`)
			}
			sb.WriteString(">")
			writeRandomFoldChildren(rng, sb, depth+1)
			sb.WriteString("</" + name + ">")
		}
	}
}

// minOpLimit returns the smallest Evaluator.OpLimit at which expr evaluates
// from node without ErrOpLimit. The op count of one evaluation is fixed, so
// the search is monotonic.
func minOpLimit(t *testing.T, eval xpath1.Evaluator, expr *xpath1.Expression, node helium.Node) int {
	t.Helper()
	lo, hi := 1, 1<<20
	_, err := eval.OpLimit(hi).Evaluate(t.Context(), expr, node)
	require.NoError(t, err, expr.String())
	for lo < hi {
		mid := (lo + hi) / 2
		_, err := eval.OpLimit(mid).Evaluate(t.Context(), expr, node)
		if errors.Is(err, xpath1.ErrOpLimit) {
			lo = mid + 1
			continue
		}
		require.NoError(t, err, expr.String())
		hi = mid
	}
	return lo
}

// pathExprReference rewrites the first `//` of src into a path expression,
// `(prefix/descendant-or-self::node())/rest`. It runs the same two walks as
// the unfolded `prefix//rest` and charges the same ops, and Compile does not
// fold across the parentheses.
func pathExprReference(src string) string {
	before, after, _ := strings.Cut(src, "//")
	return "(" + before + "/descendant-or-self::node())/" + after
}

const foldShapeDoc = `<!DOCTYPE root [<!ATTLIST a id ID #IMPLIED>]>` +
	`<root xmlns:p="urn:p"><a id="x"><b x="1">t</b><!--c--></a><p:a><b/><?pi d?></p:a>` +
	`<x><y/><x><y/></x></x><a><b x="2" xml:lang="en"/><c/></a></root>`

func TestCompileFoldShape(t *testing.T) {
	doc := parseFoldDoc(t, foldShapeDoc)
	eval := foldEvaluator([]helium.Node{doc.DocumentElement()})

	folded := []string{
		"//x", "//*", "//p:a", "//b[@x='1']", "//a[b]", "//a[not(b)]",
		"//b[normalize-space()]", "//a[@id][b]", "//a[@id or b and c]", "//b[string(.)]",
		"//b[id('x')]", "//x[y | z]", "//a[count(b) > 0]", "//b[lang('en')]", "//b[true()]",
		"//*[local-name() = 'a']", "//.", ".//b", "//self::b", "//self::node()",
		"//descendant::b", "//descendant::node()", "//descendant-or-self::b",
		"//descendant-or-self::comment()", "$nodes//b", "/root//x/y", "//a[.//b]",
		"//node()", "//text()", "//comment()", "//processing-instruction()",
		"//processing-instruction('pi')", "//comment()[string(.)]",
	}
	kept := []string{
		"//a[1]", "//a[last()]", "//a[position()=1]", "//b[position() > 1 and @x]",
		"//b[boolean(position())]", "//a[not(last())]", "//b[concat(position(), '')]",
		"//a[$nodes]", "//b[@x = $one]", "//b[ext:g()]", "//b[f()]", "//a[count(b)]",
		"//a[sum(b)]", "//x[number(.)]", "//a[-1]", "//a[1 + 1]", "//b[string-length()]",
		"//a[b[1]]", "//a[b[last()]]", "//a[(b)[1]]", "//a[.//b[2]]", "//..", "//@x",
		"//following-sibling::b", "//node()[1]", "//text()[last()]",
	}

	for _, src := range folded {
		t.Run("folds "+src, func(t *testing.T) {
			got, unfolded := foldOpLimits(t, eval, doc, src)
			require.Less(t, got, unfolded)
		})
	}
	for _, src := range kept {
		t.Run("keeps "+src, func(t *testing.T) {
			got, unfolded := foldOpLimits(t, eval, doc, src)
			require.Equal(t, unfolded, got)
		})
	}
}

// foldOpLimits returns the smallest op limits at which src and its
// pathExprReference pass from doc.
func foldOpLimits(t *testing.T, eval xpath1.Evaluator, doc *helium.Document, src string) (int, int) {
	t.Helper()
	compiled, err := xpath1.Compile(src)
	require.NoError(t, err)
	reference, err := xpath1.Compile(pathExprReference(src))
	require.NoError(t, err)
	return minOpLimit(t, eval, compiled, doc), minOpLimit(t, eval, reference, doc)
}

func TestParseKeepsDescendantOrSelfStep(t *testing.T) {
	parsed, err := xpath1.Parse("//x[y]")
	require.NoError(t, err)
	lp, ok := parsed.(*xpath1.LocationPath)
	require.True(t, ok)
	require.True(t, lp.Absolute)
	require.Len(t, lp.Steps, 2)
	require.Equal(t, xpath1.AxisDescendantOrSelf, lp.Steps[0].Axis)
	require.Equal(t, xpath1.TypeTest{Type: xpath1.NodeTestNode}, lp.Steps[0].NodeTest)
	require.Empty(t, lp.Steps[0].Predicates)
	require.Equal(t, xpath1.AxisChild, lp.Steps[1].Axis)
	require.Equal(t, xpath1.NameTest{Local: "x"}, lp.Steps[1].NodeTest)
	require.Len(t, lp.Steps[1].Predicates, 1)

	compiled, err := xpath1.Compile("//x[y]")
	require.NoError(t, err)
	require.Equal(t, "//x[y]", compiled.String())
}

// TestCompileFoldOpCount pins the one declared behavior change. The unfolded
// `//x` walks the tree twice, 2N+1 ops for the N nodes below the document;
// the folded descendant::x walks it once, N ops. An op limit between the two
// passes now, where the unfolded path (run here as a path expression Compile
// does not fold) fails.
func TestCompileFoldOpCount(t *testing.T) {
	doc := parseFoldDoc(t, `<root><x/><x/><x/></root>`)
	const nodes = 4 // root and three x elements

	compiled, err := xpath1.Compile("//x")
	require.NoError(t, err)
	unfolded, err := xpath1.Compile(pathExprReference("//x"))
	require.NoError(t, err)

	r, err := xpath1.NewEvaluator().OpLimit(nodes).Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 3)

	_, err = xpath1.NewEvaluator().OpLimit(nodes-1).Evaluate(t.Context(), compiled, doc)
	require.ErrorIs(t, err, xpath1.ErrOpLimit)

	_, err = xpath1.NewEvaluator().OpLimit(2*nodes).Evaluate(t.Context(), unfolded, doc)
	require.ErrorIs(t, err, xpath1.ErrOpLimit)

	r, err = xpath1.NewEvaluator().OpLimit(2*nodes+1).Evaluate(t.Context(), unfolded, doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 3)
}

// TestCompileFoldSeesMutation checks that a compiled `//item` reflects DOM
// changes made between evaluations.
func TestCompileFoldSeesMutation(t *testing.T) {
	doc := parseFoldDoc(t, `<root><item/><item/></root>`)
	compiled, err := xpath1.Compile("//item")
	require.NoError(t, err)

	r, err := compiled.Evaluate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)

	extra, err := doc.CreateElement("item")
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(extra))
	r, err = compiled.Evaluate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 3)
	require.Same(t, extra, r.NodeSet[2])

	helium.UnlinkNode(extra)
	r, err = compiled.Evaluate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

// TestCompileFoldHandBuiltDTDElement runs the differential test on a DTD
// built with DTD.AddChild to hold an element and a comment before an entity
// declaration, with a reference to that entity in the document. The
// descendant walk stops at the entity reference's Entity child, as the child
// axis does, so the folded and unfolded paths never reach the DTD's element
// from the entity reference.
func TestCompileFoldHandBuiltDTDElement(t *testing.T) {
	doc := parseFoldDoc(t, `<a/>`)
	dtd, err := doc.CreateInternalSubset("a", "", "")
	require.NoError(t, err)
	x, err := doc.CreateElement("x")
	require.NoError(t, err)
	require.NoError(t, dtd.AddChild(x))
	require.NoError(t, dtd.AddChild(doc.CreateComment([]byte("dtdc"))))
	_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "v")
	require.NoError(t, err)
	ref, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(ref))

	folded, err := xpath1.MustCompile(".//x").Evaluate(t.Context(), ref)
	require.NoError(t, err)
	require.Empty(t, folded.NodeSet)

	eval := foldEvaluator([]helium.Node{doc.DocumentElement()})
	contexts := foldContexts(t, doc)
	require.Len(t, contexts, 4)
	requireFoldMatchesReference(t, eval, "hand-built DTD", contexts)
}

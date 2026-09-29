package xpath1

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// differentialDocs are the fixed documents of the Compile-versus-Parse
// differential test: nested same-name elements, attributes and namespaces,
// comments and PIs, and an entity reference whose internal subset holds a
// comment and a PI before the entity declaration.
var differentialDocs = []struct {
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

// differentialExprs mixes the shapes Compile rewrites with the shapes it must
// keep, so the differential test covers both sides of the rewrite.
var differentialExprs = []string{
	// Paths the rewrite does not touch.
	"//a", "//a/b", "//a/..", "//a/parent::*", "//a/@*", "//a/namespace::*",
	"//*/self::b", "//b/following-sibling::*", "//b/preceding-sibling::*",
	"ancestor::*", "ancestor-or-self::node()", "preceding::*", "following::*",
	"//a | //b", "(//a)[2]/b", "//a[1]", "//a[b][2]", "id('x')/..", "/a/b/c",
	"/a/b/@id", "/a/*/namespace::*", "//*[@id]/ancestor::*", "$nodes/b", "//.", ".//x",

	// Paths the rewrite collapses.
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

	// Paths the rewrite must keep, because the predicate may observe position.
	"//a[last()]", "//a[position()=1]", "//b[position() mod 2 = 0]", "//b[last() - 1]",
	"//b[2]", "//a[$nodes]", "//a[count(b)]", "//a[-1]", "//b[string-length()]",
	"//a[b[1]]", "//..", "descendant-or-self::node()[1]/b", "//a[1]/b",
	"//b[f()]", "//b[ext:g()]", "//a[(b)[1]]", "//x[number(.)]", "//x[1 + 0]",
}

// positionIsOne is a custom function that reads the context position, so
// rewriting a predicate that calls it would change the selected nodes.
func positionIsOne(ctx context.Context, _ []*Result) (*Result, error) {
	fctx := GetFunctionContext(ctx)
	return &Result{Type: BooleanResult, Bool: fctx.Position() == 1}, nil
}

// differentialEvaluator binds everything the expressions above reference:
// the p and ext prefixes, two position-reading custom functions, and $nodes,
// a node-set spanning two documents.
func differentialEvaluator(nodes []helium.Node) Evaluator {
	return NewEvaluator().
		Namespaces(map[string]string{"p": "urn:p", "ext": "urn:ext"}).
		Function("f", FunctionFunc(positionIsOne)).
		FunctionNS("urn:ext", "g", FunctionFunc(positionIsOne)).
		Variables(map[string]any{"nodes": nodes})
}

func parseDifferentialDoc(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
	require.NoError(t, err)
	return doc
}

// firstNodeOf returns the first node expr selects from doc, or nil.
func firstNodeOf(t *testing.T, doc *helium.Document, expr string) helium.Node {
	t.Helper()
	r, err := Evaluate(t.Context(), doc, expr)
	require.NoError(t, err)
	require.Equal(t, NodeSetResult, r.Type)
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

type differentialContext struct {
	name string
	node helium.Node
}

// differentialContexts returns the document, a mid-document element, an
// attribute, a namespace node, an entity reference, the entity it refers to,
// and the DTD of doc, skipping any kind the document does not have. XPath
// never selects the last three, but a caller can pass any node as the
// context node, and xmldsig1 evaluates its XPath filter on every node of a
// subtree, entity references and entities included.
func differentialContexts(t *testing.T, doc *helium.Document) []differentialContext {
	t.Helper()
	candidates := []differentialContext{
		{name: "document", node: doc},
		{name: "element", node: firstNodeOf(t, doc, "/*/*[1]")},
		{name: "attribute", node: firstNodeOf(t, doc, "//@*")},
		{name: "namespace", node: firstNodeOf(t, doc, "/*/*[last()]/namespace::*[last()]")},
		{name: "entity reference", node: firstEntityRef(doc)},
	}
	if ref := firstEntityRef(doc); ref != nil {
		candidates = append(candidates, differentialContext{name: "entity", node: ref.FirstChild()})
	}
	if dtd := doc.IntSubset(); dtd != nil {
		candidates = append(candidates, differentialContext{name: "dtd", node: dtd})
	}
	var out []differentialContext
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
func requireSameResult(t *testing.T, label string, want *Result, wantErr error, got *Result, gotErr error) {
	t.Helper()
	if wantErr != nil || gotErr != nil {
		require.Error(t, wantErr, label)
		require.Error(t, gotErr, label)
		require.Equal(t, wantErr.Error(), gotErr.Error(), label)
		return
	}
	require.Equal(t, want.Type, got.Type, label)
	switch want.Type {
	case NodeSetResult:
		require.Len(t, got.NodeSet, len(want.NodeSet), label)
		for i := range want.NodeSet {
			require.True(t, sameNode(want.NodeSet[i], got.NodeSet[i]), "%s: node %d differs", label, i)
		}
	case NumberResult:
		if math.IsNaN(want.Number) {
			require.True(t, math.IsNaN(got.Number), label)
			return
		}
		require.Equal(t, want.Number, got.Number, label)
	case BooleanResult:
		require.Equal(t, want.Bool, got.Bool, label)
	case StringResult:
		require.Equal(t, want.String, got.String, label)
	}
}

// requireCompileMatchesParse evaluates every differential expression twice
// from each context node: once as Compile builds it, and once from the
// unrewritten Parse AST. Both must give the same result in the same order.
func requireCompileMatchesParse(t *testing.T, eval Evaluator, docName string, contexts []differentialContext) {
	t.Helper()
	for _, src := range differentialExprs {
		compiled, err := Compile(src)
		require.NoError(t, err, src)
		ast, err := Parse(src)
		require.NoError(t, err, src)
		plain := &Expression{source: src, ast: ast}
		for _, c := range contexts {
			label := fmt.Sprintf("%s / %s / %s", docName, c.name, src)
			want, wantErr := eval.Evaluate(t.Context(), plain, c.node)
			got, gotErr := eval.Evaluate(t.Context(), compiled, c.node)
			requireSameResult(t, label, want, wantErr, got, gotErr)
		}
	}
}

func TestCompileRewriteMatchesParse(t *testing.T) {
	docs := make([]*helium.Document, len(differentialDocs))
	for i, d := range differentialDocs {
		docs[i] = parseDifferentialDoc(t, d.xml)
	}
	// $nodes spans two documents.
	nodes := []helium.Node{docs[0].DocumentElement(), docs[1].DocumentElement()}
	eval := differentialEvaluator(nodes)

	for i, d := range differentialDocs {
		t.Run(d.name, func(t *testing.T) {
			contexts := differentialContexts(t, docs[i])
			require.Len(t, contexts, d.contexts)
			requireCompileMatchesParse(t, eval, d.name, contexts)
		})
	}

	t.Run("random documents", func(t *testing.T) {
		rng := rand.New(rand.NewPCG(1, 2))
		for i := range 40 {
			doc := parseDifferentialDoc(t, randomDifferentialDoc(rng))
			name := fmt.Sprintf("random %d", i)
			requireCompileMatchesParse(t, eval, name, differentialContexts(t, doc))
		}
	})
}

// randomDifferentialDoc builds a small random document from the element
// names the differential expressions use, with attributes, text, comments
// and PIs mixed in.
func randomDifferentialDoc(rng *rand.Rand) string {
	var sb strings.Builder
	sb.WriteString(`<root xmlns:p="urn:p">`)
	writeRandomChildren(rng, &sb, 0)
	sb.WriteString(`</root>`)
	return sb.String()
}

var randomElementNames = []string{"a", "b", "c", "x", "y", "p:a"}

func writeRandomChildren(rng *rand.Rand, sb *strings.Builder, depth int) {
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
			name := randomElementNames[rng.IntN(len(randomElementNames))]
			sb.WriteString("<" + name)
			if rng.IntN(2) == 0 {
				fmt.Fprintf(sb, ` x="%d"`, rng.IntN(2))
			}
			if rng.IntN(3) == 0 {
				sb.WriteString(` id="x"`)
			}
			sb.WriteString(">")
			writeRandomChildren(rng, sb, depth+1)
			sb.WriteString("</" + name + ">")
		}
	}
}

// renderAST prints an AST in unabbreviated XPath syntax, so a test can state
// the exact shape Compile produced.
func renderAST(e Expr) string {
	switch v := e.(type) {
	case *LocationPath:
		return renderLocationPath(v)
	case BinaryExpr:
		return "(" + renderAST(v.Left) + " " + v.Op.String() + " " + renderAST(v.Right) + ")"
	case UnaryExpr:
		return "-" + renderAST(v.Operand)
	case LiteralExpr:
		return "'" + v.Value + "'"
	case NumberExpr:
		return fmt.Sprintf("%g", v.Value)
	case VariableExpr:
		return "$" + v.Name
	case FunctionCall:
		args := make([]string, len(v.Args))
		for i, arg := range v.Args {
			args[i] = renderAST(arg)
		}
		name := v.Name
		if v.Prefix != "" {
			name = v.Prefix + ":" + name
		}
		return name + "(" + strings.Join(args, ", ") + ")"
	case FilterExpr:
		return "(" + renderAST(v.Expr) + ")" + renderPredicates(v.Predicates)
	case UnionExpr:
		return renderAST(v.Left) + " | " + renderAST(v.Right)
	case PathExpr:
		return renderAST(v.Filter) + "/" + renderLocationPath(v.Path)
	}
	return fmt.Sprintf("%T", e)
}

func renderLocationPath(lp *LocationPath) string {
	steps := make([]string, len(lp.Steps))
	for i, s := range lp.Steps {
		steps[i] = s.Axis.String() + "::" + renderNodeTest(s.NodeTest) + renderPredicates(s.Predicates)
	}
	out := strings.Join(steps, "/")
	if lp.Absolute {
		return "/" + out
	}
	return out
}

func renderNodeTest(nt NodeTest) string {
	switch v := nt.(type) {
	case NameTest:
		if v.Prefix != "" {
			return v.Prefix + ":" + v.Local
		}
		return v.Local
	case TypeTest:
		switch v.Type {
		case NodeTestText:
			return "text()"
		case NodeTestComment:
			return "comment()"
		case NodeTestProcessingInstruction:
			return "processing-instruction()"
		}
		return "node()"
	case PITest:
		return "processing-instruction('" + v.Target + "')"
	}
	return fmt.Sprintf("%T", nt)
}

func renderPredicates(preds []Expr) string {
	var sb strings.Builder
	for _, p := range preds {
		sb.WriteString("[" + renderAST(p) + "]")
	}
	return sb.String()
}

func TestCompileCollapsesDescendantSteps(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		{"//x", "/descendant::x"},
		{"//*", "/descendant::*"},
		{"//p:x", "/descendant::p:x"},
		{"//x[@a='v']", "/descendant::x[(attribute::a = 'v')]"},
		{"//x[y]", "/descendant::x[child::y]"},
		{"//x[not(z)]", "/descendant::x[not(child::z)]"},
		{"//x[normalize-space()]", "/descendant::x[normalize-space()]"},
		{"//x[@a][y]", "/descendant::x[attribute::a][child::y]"},
		{"//x[@a or y and z]", "/descendant::x[(attribute::a or (child::y and child::z))]"},
		{"//x[string(.)]", "/descendant::x[string(self::node())]"},
		{"//x[id('a')]", "/descendant::x[id('a')]"},
		{"//x[y | z]", "/descendant::x[child::y | child::z]"},
		{"//x[count(y) > 1]", "/descendant::x[(count(child::y) > 1)]"},
		{"//x[lang('en')]", "/descendant::x[lang('en')]"},
		{"//x[true()]", "/descendant::x[true()]"},
		{"//x[local-name() = 'x']", "/descendant::x[(local-name() = 'x')]"},
		{"//.", "/descendant-or-self::node()"},
		{".//x", "self::node()/descendant::x"},
		{"//x//y", "/descendant::x/descendant::y"},
		{"//.//x", "/descendant::x"},
		{"//self::x", "/descendant-or-self::x"},
		{"//self::node()", "/descendant-or-self::node()"},
		{"//descendant::x", "/descendant::x"},
		{"//descendant::node()", "/descendant::node()"},
		{"//descendant-or-self::x", "/descendant-or-self::x"},
		{"//descendant-or-self::comment()", "/descendant-or-self::comment()"},
		{"descendant-or-self::node()/child::x", "descendant::x"},
		{"$v//x", "$v/descendant::x"},
		{"count(//x)", "count(/descendant::x)"},
		{"//a[.//b]", "/descendant::a[self::node()/descendant::b]"},
		{"(//a)[1]//b", "(/descendant::a)[1]/descendant::b"},
		{"/r//x/y", "/child::r/descendant::x/child::y"},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			compiled, err := Compile(tc.expr)
			require.NoError(t, err)
			require.Equal(t, tc.want, renderAST(compiled.ast))
		})
	}
}

func TestCompileKeepsPositionalSteps(t *testing.T) {
	// Each expression keeps its descendant-or-self::node() step: the predicate
	// may observe the context position or size, the step's node test can
	// match a DTD comment or PI, or the step cannot fold at all.
	exprs := []string{
		"//x[1]",
		"//x[last()]",
		"//x[position()=1]",
		"//x[position() > 1 and @a]",
		"//x[boolean(position())]",
		"//x[not(last())]",
		"//x[concat(position(), '')]",
		"//x[$v]",
		"//x[@a = $v]",
		"//x[ext:f()]",
		"//x[f()]",
		"//x[count(y)]",
		"//x[sum(y)]",
		"//x[number(.)]",
		"//x[-1]",
		"//x[1 + 1]",
		"//x[string-length()]",
		"//x[a[1]]",
		"//x[y[last()]]",
		"//x[(y)[1]]",
		"//x[.//y[2]]",
		"//..",
		"//@a",
		"//following-sibling::x",
		"//node()",
		"//text()",
		"//comment()",
		"//processing-instruction()",
		"//processing-instruction('t')",
		"descendant-or-self::node()[1]/x",
		"descendant-or-self::*/x",
	}
	for _, src := range exprs {
		t.Run(src, func(t *testing.T) {
			compiled, err := Compile(src)
			require.NoError(t, err)
			parsed, err := Parse(src)
			require.NoError(t, err)
			require.Equal(t, renderAST(parsed), renderAST(compiled.ast))
			require.Contains(t, renderAST(compiled.ast), "descendant-or-self::")
		})
	}
}

func TestParseKeepsDescendantOrSelfStep(t *testing.T) {
	parsed, err := Parse("//x[y]")
	require.NoError(t, err)
	require.Equal(t, "/descendant-or-self::node()/child::x[child::y]", renderAST(parsed))

	compiled, err := Compile("//x[y]")
	require.NoError(t, err)
	require.Equal(t, "//x[y]", compiled.String())
}

func TestBuiltinResultTypesCoverBuiltins(t *testing.T) {
	require.Len(t, builtinResultTypes, len(builtinFunctions))
	for name := range builtinFunctions {
		typ, ok := builtinResultTypes[name]
		require.True(t, ok, name)
		require.NotEqual(t, staticUnknown, typ, name)
	}
}

// TestCompileCollapsedOpCount pins the one behavior the rewrite changes. The
// unrewritten `//x` walks the tree twice: 2N+1 operations for N nodes below
// the document. The collapsed descendant::x walks it once: N operations. An
// op limit between the two now succeeds where the unrewritten path fails.
func TestCompileCollapsedOpCount(t *testing.T) {
	doc := parseDifferentialDoc(t, `<root><x/><x/><x/></root>`)
	const nodes = 4 // root and three x elements

	ast, err := Parse("//x")
	require.NoError(t, err)
	unrewritten := &Expression{source: "//x", ast: ast}
	compiled, err := Compile("//x")
	require.NoError(t, err)

	r, err := NewEvaluator().OpLimit(nodes).Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 3)

	_, err = NewEvaluator().OpLimit(nodes-1).Evaluate(t.Context(), compiled, doc)
	require.ErrorIs(t, err, ErrOpLimit)

	_, err = NewEvaluator().OpLimit(2*nodes).Evaluate(t.Context(), unrewritten, doc)
	require.ErrorIs(t, err, ErrOpLimit)

	r, err = NewEvaluator().OpLimit(2*nodes+1).Evaluate(t.Context(), unrewritten, doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 3)
}

// TestCompileCollapsedSeesMutation checks that a compiled `//item` reflects
// DOM changes made between evaluations.
func TestCompileCollapsedSeesMutation(t *testing.T) {
	doc := parseDifferentialDoc(t, `<root><item/><item/></root>`)
	compiled, err := Compile("//item")
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

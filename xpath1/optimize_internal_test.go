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
		contexts: 4,
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
		contexts: 5,
		xml: `<!DOCTYPE a [<!--subset comment--><?subset pi?><!ENTITY e "<b>ent</b>">` +
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
	"//processing-instruction('pi')", "//@*", "//p:a", "//*[@p:y]",
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
// attribute, a namespace node, and an entity reference of doc, skipping any
// kind the document does not have.
func differentialContexts(t *testing.T, doc *helium.Document) []differentialContext {
	t.Helper()
	candidates := []differentialContext{
		{name: "document", node: doc},
		{name: "element", node: firstNodeOf(t, doc, "/*/*[1]")},
		{name: "attribute", node: firstNodeOf(t, doc, "//@*")},
		{name: "namespace", node: firstNodeOf(t, doc, "/*/*[last()]/namespace::*[last()]")},
		{name: "entity reference", node: firstEntityRef(doc)},
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

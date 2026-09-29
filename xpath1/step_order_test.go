package xpath1_test

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// stepOrderGoldenPath holds the recorded result of every case in
// TestStepResultOrder. Set XPATH1_UPDATE_STEP_ORDER=1 to rewrite it.
var stepOrderGoldenPath = filepath.Join("testdata", "step_order.golden")

type stepOrderDoc struct {
	name  string
	src   string
	subst bool
}

var stepOrderDocs = []stepOrderDoc{
	{
		name: "nested",
		src:  `<a id="r"><b id="b1"><a id="a2"><b id="b2"/></a></b><b id="b3"><a id="a3"><b id="b4"><c/></b></a><c/></b></a>`,
	},
	{
		name: "mixed",
		src: `<?xml version="1.0"?><!--lead--><?lead x?>` +
			`<a xmlns:p="urn:p" xmlns:q="urn:q" id="x" p:k="1">` +
			`<b id="b1">t1<a id="a2"><b id="b2" p:k="2"/><c/></a><!--c1--></b>` +
			`<b id="b3" xmlns:p="urn:p2"><x/><?pi y?>t2<c id="c1"/></b>` +
			`<p:x id="px"><x p:r="2"/><b/></p:x><c/></a><!--trail-->`,
	},
	{
		name: "entity",
		src: `<!DOCTYPE a [<!ENTITY e "<b id='eb'>ent<!--ec--></b>"><!--dtdc--><?dtdpi z?>]>` +
			`<a xmlns:p="urn:p"><b id="b1"/>&e;<!--c--><x id="x1">&e;</x><b id="b2"/></a>`,
	},
	{
		name: "entity-subst",
		src: `<!DOCTYPE a [<!ENTITY e "<b id='eb'>ent<!--ec--></b>"><!--dtdc--><?dtdpi z?>]>` +
			`<a xmlns:p="urn:p"><b id="b1"/>&e;<!--c--><x id="x1">&e;</x><b id="b2"/></a>`,
		subst: true,
	},
}

// stepOrderOtherDoc is bound, together with the document under test, to the
// $nodes and $other variables so cross-document node-sets are covered.
const stepOrderOtherDoc = `<a id="o"><b id="o1"/><b id="o2"><c/></b><c/></a>`

var stepOrderContexts = []struct {
	name string
	expr string
}{
	{name: "doc", expr: ""},
	{name: "elem", expr: "(//*[@id])[2]"},
	{name: "attr", expr: "(//@id)[1]"},
	{name: "ns", expr: "/*/namespace::p"},
}

var stepOrderExprs = []string{
	"//a", "//a/b", "//a/..", "//a/parent::*", "//a/@*", "//a/namespace::*",
	"//*/self::b", "//b/following-sibling::*", "//b/preceding-sibling::*",
	"ancestor::*", "ancestor-or-self::node()", "preceding::*", "following::*",
	"preceding::node()", "following::node()", "preceding-sibling::node()", "following-sibling::node()",
	"//a | //b", "(//a)[2]/b", "//a[1]", "//a[b][2]", "id('x')/..", "id('b1 b3')/*",
	"/a/b/c", "/a/b/@id", "/a/*/namespace::*", "//*[@id]/ancestor::*", "//.", ".//x",
	"//node()", "//comment()", "//text()", "//processing-instruction()", "descendant::node()",
	"..", "//*/..", "//@*/..", "//@*", "//namespace::*", "//*/namespace::*/..",
	"//b/c", "//b/following::*", "//b/preceding::*", "//b/ancestor::*", "//b/descendant::*",
	"//x/ancestor-or-self::*/@id", "/a/b/node()", "/a/node()/node()", "//b/node()/..",
	"/descendant::b/ancestor::a", "//b[1]/following-sibling::node()[1]", "//c/preceding::node()[1]",
	"/*/*/*", "/*/*/*/..", "/*/*/@*", "//comment()/..", "//b/parent::node()/parent::node()",
	"$nodes/b", "$nodes//b", "$nodes/..", "/a/b | $other/b", "//b | $nodes", "//b | $other//b",
	"$nodes | //b", "//b | ($other/b | $other/c)", "$other/b | /a/b", "($nodes)[1]//c",
}

func parseStepOrderDoc(t testing.TB, src string, subst bool) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().SubstituteEntities(subst).Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	return doc
}

// describeNode renders n as a path from its root that names every step by
// node kind and raw sibling index, so any two distinct nodes (including
// nodes outside the XPath data model) describe differently.
func describeNode(n helium.Node, labels map[helium.Node]string) string {
	switch n.Type() {
	case helium.NamespaceNode:
		return describeNode(n.Parent(), labels) + "/namespace::" + n.Name()
	case helium.AttributeNode:
		return describeNode(n.Parent(), labels) + "/@" + n.Name()
	case helium.DocumentNode:
		if l, ok := labels[n]; ok {
			return l
		}
		return "doc?"
	}
	step := nodeKind(n)
	parent := n.Parent()
	if parent == nil {
		return "orphan:" + step
	}
	return describeNode(parent, labels) + "/" + step + "[" + siblingIndex(parent, n) + "]"
}

func nodeKind(n helium.Node) string {
	switch n.Type() {
	case helium.ElementNode:
		return n.Name()
	case helium.TextNode:
		return "text()"
	case helium.CommentNode:
		return "comment()"
	case helium.ProcessingInstructionNode:
		return "pi(" + n.Name() + ")"
	case helium.EntityRefNode:
		return "entref(" + n.Name() + ")"
	case helium.DTDNode:
		return "dtd()"
	}
	return "type" + strconv.Itoa(int(n.Type())) + "(" + n.Name() + ")"
}

func siblingIndex(parent, n helium.Node) string {
	i := 0
	for c := parent.FirstChild(); c != nil && i < 100000; c = c.NextSibling() {
		if c == n {
			return strconv.Itoa(i)
		}
		i++
	}
	return "?"
}

type stepOrderFixture struct {
	doc    *helium.Document
	other  *helium.Document
	labels map[helium.Node]string
	eval   xpath1.Evaluator
}

func newStepOrderFixture(t testing.TB, d stepOrderDoc) stepOrderFixture {
	t.Helper()
	doc := parseStepOrderDoc(t, d.src, d.subst)
	other := parseStepOrderDoc(t, stepOrderOtherDoc, false)
	labels := map[helium.Node]string{doc: "d1:", other: "d2:"}
	vars := map[string]any{
		"nodes": []helium.Node{other.DocumentElement(), doc.DocumentElement()},
		"other": []helium.Node{other.DocumentElement()},
	}
	return stepOrderFixture{
		doc:    doc,
		other:  other,
		labels: labels,
		eval:   xpath1.NewEvaluator().Variables(vars),
	}
}

// contextNode resolves a context by evaluating expr from the document node;
// the first node of the result is the context. It returns nil when the
// expression selects nothing in this document.
func (f stepOrderFixture) contextNode(t testing.TB, expr string) helium.Node {
	t.Helper()
	if expr == "" {
		return f.doc
	}
	r, err := f.eval.Evaluate(t.Context(), xpath1.MustCompile(expr), f.doc)
	require.NoError(t, err)
	if len(r.NodeSet) == 0 {
		return nil
	}
	return r.NodeSet[0]
}

// describeResult evaluates expr and renders its node-set. It also checks that
// the node-set is already sorted and duplicate-free according to a fresh
// document-order index (the order oracle).
func (f stepOrderFixture) describeResult(t testing.TB, ctxNode helium.Node, expr string) string {
	t.Helper()
	r, err := f.eval.Evaluate(t.Context(), xpath1.MustCompile(expr), ctxNode)
	if err != nil {
		return "ERR " + err.Error()
	}
	require.Equal(t, xpath1.NodeSetResult, r.Type, expr)
	sorted, err := ixpath.DeduplicateNodes(slices.Clone(r.NodeSet), &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Equal(t, r.NodeSet, sorted, "result of %q is not in document order", expr)
	parts := make([]string, len(r.NodeSet))
	for i, n := range r.NodeSet {
		parts[i] = describeNode(n, f.labels)
	}
	return strings.Join(parts, " ")
}

// TestStepResultOrder pins the node-set and its order for a matrix of
// documents, context nodes and location paths against a recorded golden
// file, so any change to how a step orders or deduplicates its result shows
// up as a diff.
func TestStepResultOrder(t *testing.T) {
	var got strings.Builder
	for _, d := range stepOrderDocs {
		f := newStepOrderFixture(t, d)
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr)
			if ctxNode == nil {
				continue
			}
			for _, expr := range stepOrderExprs {
				fmt.Fprintf(&got, "%s|%s|%s => %s\n", d.name, c.name, expr, f.describeResult(t, ctxNode, expr))
			}
		}
	}

	if os.Getenv("XPATH1_UPDATE_STEP_ORDER") == "1" {
		require.NoError(t, os.MkdirAll(filepath.Dir(stepOrderGoldenPath), 0o755))
		require.NoError(t, os.WriteFile(stepOrderGoldenPath, []byte(got.String()), 0o644))
		return
	}
	want, err := os.ReadFile(stepOrderGoldenPath)
	require.NoError(t, err)
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(got.String(), "\n")
	require.Len(t, gotLines, len(wantLines))
	for i := range wantLines {
		require.Equal(t, wantLines[i], gotLines[i])
	}
}

// TestStepResultOrderRandom runs the expression matrix over generated
// documents and checks every node-set against a fresh document-order index.
func TestStepResultOrderRandom(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // deterministic test input
	for range 40 {
		src := randomStepOrderDoc(rng)
		f := newStepOrderFixture(t, stepOrderDoc{name: "random", src: src})
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr)
			if ctxNode == nil {
				continue
			}
			for _, expr := range stepOrderExprs {
				f.describeResult(t, ctxNode, expr)
			}
		}
	}
}

// randomStepOrderDoc builds a small document of nested a/b/c/x elements with
// optional attributes, text, comments and namespace declarations.
func randomStepOrderDoc(rng *rand.Rand) string {
	g := randomDocGen{rng: rng}
	g.emit(0)
	return g.b.String()
}

type randomDocGen struct {
	rng *rand.Rand
	b   strings.Builder
	id  int
}

func (g *randomDocGen) emit(depth int) {
	names := []string{"a", "b", "c", "x"}
	name := names[g.rng.IntN(len(names))]
	g.b.WriteString("<" + name)
	if depth == 0 {
		g.b.WriteString(` xmlns:p="urn:p"`)
	}
	if g.rng.IntN(2) == 0 {
		g.id++
		fmt.Fprintf(&g.b, ` id="n%d"`, g.id)
	}
	if g.rng.IntN(4) == 0 {
		g.b.WriteString(` p:k="v"`)
	}
	g.b.WriteString(">")
	kids := 0
	if depth < 5 {
		kids = g.rng.IntN(4)
	}
	for range kids {
		switch g.rng.IntN(5) {
		case 0:
			g.b.WriteString("t")
		case 1:
			g.b.WriteString("<!--c-->")
		default:
			g.emit(depth + 1)
		}
	}
	g.b.WriteString("</" + name + ">")
}

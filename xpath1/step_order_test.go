package xpath1_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
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
	// build, when set, edits the parsed document before the cases run.
	build func(testing.TB, *helium.Document)
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
	{
		name: "entity-late-decl",
		src: `<!DOCTYPE a [<!--dtdc--><?dtdpi z?><!ENTITY e "<b id='eb'>ent<!--ec--></b>">]>` +
			`<a xmlns:p="urn:p"><b id="b1"/>&e;<!--c--><x id="x1">&e;</x><b id="b2"/></a>`,
	},
	{
		// f is referenced from e's content and from the document, so the
		// content of both entities sits in the order index at a reference
		// inside the document element.
		name: "entity-nested",
		src: `<!DOCTYPE a [<!--dtdc--><!ENTITY f "<c id='fc'><d/>tf</c>">` +
			`<!ENTITY e "<b id='eb'>&f;<c id='ec'/></b><x id='ex'>te</x>"><?dtdpi z?>]>` +
			`<a><b id="b1">&f;</b>&e;<x id="x1"><c id="c1"/>&f;</x>&e;<b id="b2"/></a>`,
	},
	{
		// The only reference to e sits in y, which has the same depth as
		// the top-level elements of e's content, so the order index places
		// that content inside the subtree of y.
		name: "entity-in-body",
		src: `<!DOCTYPE a [<!ENTITY e "<b id='eb'><c id='ec'/></b><x id='ex'>te</x>">]>` +
			`<a><p><y id="y1">&e;<z id="z1"/></y></p></a>`,
	},
	{
		name:  "handbuilt-dtd",
		src:   `<a xmlns:p="urn:p"><b id="b1"/><x id="x1"/></a>`,
		build: addHandBuiltDTD,
	},
}

// addHandBuiltDTD gives doc an internal subset built through the DOM API
// that holds an element, a comment and an entity declaration, in that
// order, and appends a reference to the entity to the root element. A parsed
// DTD never holds an element; DTD.AddChild accepts one.
func addHandBuiltDTD(t testing.TB, doc *helium.Document) {
	t.Helper()
	dtd, err := doc.CreateInternalSubset("a", "", "")
	require.NoError(t, err)
	x, err := doc.CreateElement("x")
	require.NoError(t, err)
	require.NoError(t, x.SetAttribute("id", "dtdx"))
	require.NoError(t, dtd.AddChild(x))
	require.NoError(t, dtd.AddChild(doc.CreateComment([]byte("dtdc"))))
	_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "v")
	require.NoError(t, err)
	ref, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(ref))
}

// stepOrderOtherDoc is bound, together with the document under test, to the
// $nodes and $other variables so cross-document node-sets are covered.
const stepOrderOtherDoc = `<a id="o"><b id="o1"/><b id="o2"><c/></b><c/></a>`

// stepOrderContexts names the context nodes of the matrix. A context is
// either resolved by evaluating expr from the document node or, when pick is
// set, picked from the tree directly: XPath never selects entity references,
// entities or the DTD, but a caller can pass any node as the context node.
var stepOrderContexts = []struct {
	name string
	expr string
	pick func(*helium.Document) helium.Node
}{
	{name: "doc", expr: ""},
	{name: "elem", expr: "(//*[@id])[2]"},
	{name: "attr", expr: "(//@id)[1]"},
	{name: "ns", expr: "/*/namespace::p"},
	{name: "entref", pick: pickEntityRef},
	{name: "entity", pick: pickEntity},
	{name: "dtd", pick: pickDTD},
	{name: "entelem", pick: pickEntityElem},
	{name: "enttext", pick: pickEntityText},
}

// entityContent returns every node inside the parsed content of the
// entities declared in the internal subset of doc, in raw traversal order.
func entityContent(doc *helium.Document) []helium.Node {
	dtd := doc.IntSubset()
	if dtd == nil {
		return nil
	}
	var nodes []helium.Node
	for decl := range helium.Children(dtd) {
		if decl.Type() != helium.EntityNode {
			continue
		}
		for n := range helium.Descendants(decl) {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// pickEntityElem returns the first element inside entity content, or nil.
// XPath never reaches entity content from the document, but a caller can
// pass such a node as the context node.
func pickEntityElem(doc *helium.Document) helium.Node {
	for _, n := range entityContent(doc) {
		if n.Type() == helium.ElementNode {
			return n
		}
	}
	return nil
}

// pickEntityText returns the first text node inside entity content, or nil.
func pickEntityText(doc *helium.Document) helium.Node {
	for _, n := range entityContent(doc) {
		if n.Type() == helium.TextNode {
			return n
		}
	}
	return nil
}

// entityElems returns the elements inside entity content of doc, sorted and
// deduplicated by a fresh document-order index, the way a caller would bind
// a node-set drawn from the content of several entities.
func entityElems(t testing.TB, doc *helium.Document) []helium.Node {
	t.Helper()
	var elems []helium.Node
	for _, n := range entityContent(doc) {
		if n.Type() == helium.ElementNode {
			elems = append(elems, n)
		}
	}
	elems, err := ixpath.DeduplicateNodes(elems, &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	return elems
}

// pickEntityRef returns the first entity reference of doc, or nil.
func pickEntityRef(doc *helium.Document) helium.Node {
	return firstEntityRef(doc)
}

// pickEntity returns the Entity node the first entity reference of doc
// refers to, or nil.
func pickEntity(doc *helium.Document) helium.Node {
	ref := firstEntityRef(doc)
	if ref == nil {
		return nil
	}
	return ref.FirstChild()
}

// pickDTD returns the internal subset of doc, or nil.
func pickDTD(doc *helium.Document) helium.Node {
	dtd := doc.IntSubset()
	if dtd == nil {
		return nil
	}
	return dtd
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
	"descendant-or-self::node()", ".//node()", ".//comment()", ".//processing-instruction()", ".//text()",
	"descendant::node()/..", "child::node()", "self::node()", "descendant::x",
	"$ents/node()", "$ents/*/node()", "$ents/..", "$ents/self::*", "$ents/@*", "$ents/namespace::*",
	"$ents/following-sibling::node()", "$ents/following::node()", "$ents/preceding::node()",
	"$ents/descendant::node()", "$ents/ancestor::node()", "$ents | //b", "$ents | $ents/node()",
	"following::*[count(ancestor::node()) = 3]", "following::*[count(ancestor::node()) = 3]/node()",
	"following::*[count(ancestor::node()) = 3]/@*", "following::*[count(ancestor::node()) = 3]/node()/..",
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
	if d.build != nil {
		d.build(t, doc)
	}
	other := parseStepOrderDoc(t, stepOrderOtherDoc, false)
	labels := map[helium.Node]string{doc: "d1:", other: "d2:"}
	vars := map[string]any{
		"nodes": []helium.Node{other.DocumentElement(), doc.DocumentElement()},
		"other": []helium.Node{other.DocumentElement()},
		"ents":  entityElems(t, doc),
	}
	return stepOrderFixture{
		doc:    doc,
		other:  other,
		labels: labels,
		eval:   xpath1.NewEvaluator().Variables(vars),
	}
}

// contextNode resolves a context with pick when it is set, and otherwise by
// evaluating expr from the document node; the first node of the result is
// the context. It returns nil when the document has no such node.
func (f stepOrderFixture) contextNode(t testing.TB, expr string, pick func(*helium.Document) helium.Node) helium.Node {
	t.Helper()
	if pick != nil {
		return pick(f.doc)
	}
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
	got := f.describeNodes(r.NodeSet)
	if !slices.Equal(r.NodeSet, sorted) {
		// Compare the rendered paths: a diff of the node values themselves
		// prints whole trees.
		require.Equal(t, f.describeNodes(sorted), got, "result of %q is not in document order", expr)
		require.FailNow(t, "result of "+expr+" holds different nodes that render alike")
	}
	return got
}

// describeNodes renders every node of nodes with describeNode.
func (f stepOrderFixture) describeNodes(nodes []helium.Node) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
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
			ctxNode := f.contextNode(t, c.expr, c.pick)
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
			ctxNode := f.contextNode(t, c.expr, c.pick)
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

// evalIDs evaluates expr from ctxNode and returns the id attribute (or the
// element name when there is none) of every selected node, in result order.
func evalIDs(t *testing.T, ctx context.Context, eval xpath1.Evaluator, expr string, ctxNode helium.Node) []string {
	t.Helper()
	r, err := eval.Evaluate(ctx, xpath1.MustCompile(expr), ctxNode)
	require.NoError(t, err)
	ids := make([]string, 0, len(r.NodeSet))
	for _, n := range r.NodeSet {
		e, ok := n.(*helium.Element)
		if !ok {
			ids = append(ids, n.Name())
			continue
		}
		if v, ok := e.GetAttribute("id"); ok {
			ids = append(ids, v)
			continue
		}
		ids = append(ids, e.Name())
	}
	return ids
}

func moveToEnd(t *testing.T, n *helium.Element) {
	t.Helper()
	parent, ok := n.Parent().(*helium.Element)
	require.True(t, ok)
	helium.UnlinkNode(n)
	require.NoError(t, parent.AddChild(n))
}

// TestStepResultOrderMutation checks that results follow the current tree
// after a mutation, with a fresh cache per evaluation and with a
// caller-supplied cache that is Reset after the mutation.
func TestStepResultOrderMutation(t *testing.T) {
	const src = `<root><item id="i1"/><item id="i2"/><item id="i3"/><other id="o"/></root>`
	eval := xpath1.NewEvaluator()

	t.Run("fresh cache per evaluation", func(t *testing.T) {
		doc := parseStepOrderDoc(t, src, false)
		require.Equal(t, []string{"i1", "i2", "i3"}, evalIDs(t, t.Context(), eval, "//item", doc))
		first := doc.DocumentElement().FirstChild().(*helium.Element)
		moveToEnd(t, first)
		require.Equal(t, []string{"i2", "i3", "i1"}, evalIDs(t, t.Context(), eval, "//item", doc))
		require.Equal(t, []string{"i2", "i3", "o", "i1"}, evalIDs(t, t.Context(), eval, "/root/item | /root/other", doc))
		helium.UnlinkNode(first)
		require.Equal(t, []string{"i2", "i3"}, evalIDs(t, t.Context(), eval, "/root/item", doc))
	})

	t.Run("caller-supplied cache reset after the mutation", func(t *testing.T) {
		doc := parseStepOrderDoc(t, src, false)
		cache := &ixpath.DocOrderCache{}
		ctx := ixpath.WithDocOrderCache(t.Context(), cache)
		// The first evaluation only reserves the document; the second
		// indexes it through the union.
		require.Equal(t, []string{"i1", "i2", "i3"}, evalIDs(t, ctx, eval, "/root/item", doc))
		require.Equal(t, []string{"i1", "i2", "i3", "o"}, evalIDs(t, ctx, eval, "/root/item | /root/other", doc))

		moveToEnd(t, doc.DocumentElement().FirstChild().(*helium.Element))
		cache.Reset()
		require.Equal(t, []string{"i2", "i3", "i1"}, evalIDs(t, ctx, eval, "/root/item", doc))
		require.Equal(t, []string{"i2", "i3", "o", "i1"}, evalIDs(t, ctx, eval, "/root/item | /root/other", doc))
	})

	t.Run("caller-supplied cache reset while the document is only reserved", func(t *testing.T) {
		doc := parseStepOrderDoc(t, src, false)
		cache := &ixpath.DocOrderCache{}
		ctx := ixpath.WithDocOrderCache(t.Context(), cache)
		require.Equal(t, []string{"i1", "i2", "i3"}, evalIDs(t, ctx, eval, "/root/item", doc))

		moveToEnd(t, doc.DocumentElement().FirstChild().(*helium.Element))
		cache.Reset()
		require.Equal(t, []string{"i2", "i3", "o", "i1"}, evalIDs(t, ctx, eval, "/root/item | /root/other", doc))
	})
}

// TestStepResultOrderSharedCache checks a caller-supplied cache that holds a
// document reserved by a skipping step but not indexed: later evaluations
// order that document, and a second document, as they would if the first
// evaluation had indexed it.
func TestStepResultOrderSharedCache(t *testing.T) {
	doc1 := parseStepOrderDoc(t, `<a><b id="x1"><c id="xc"/></b><b id="x2"/></a>`, false)
	doc2 := parseStepOrderDoc(t, `<a><b id="y1"/><b id="y2"><c id="yc"/></b></a>`, false)
	eval := xpath1.NewEvaluator().Variables(map[string]any{
		"other": []helium.Node{doc2.DocumentElement()},
	})
	cache := &ixpath.DocOrderCache{}
	ctx := ixpath.WithDocOrderCache(t.Context(), cache)

	// Every step of /a/b skips the sort, so doc1 is reserved, not indexed.
	require.Equal(t, []string{"x1", "x2"}, evalIDs(t, ctx, eval, "/a/b", doc1))

	// doc1 was registered first, so its nodes sort before doc2's.
	require.Equal(t, []string{"x1", "x2", "y1", "y2"}, evalIDs(t, ctx, eval, "$other/b | /a/b", doc1))
	require.Equal(t, []string{"x1", "xc", "x2", "y1", "y2", "yc"}, evalIDs(t, ctx, eval, "$other//b | $other//c | /a/b/c | /a/b", doc1))
	// A lookup inside the reserved document orders it correctly.
	require.Equal(t, []string{"x1", "xc", "x2"}, evalIDs(t, ctx, eval, "/a/b/c | /a/b", doc1))

	// A second cache that registers doc2 first orders doc2 first, as the
	// registration order dictates.
	cache2 := &ixpath.DocOrderCache{}
	ctx2 := ixpath.WithDocOrderCache(t.Context(), cache2)
	require.Equal(t, []string{"y1", "y2"}, evalIDs(t, ctx2, eval, "/a/b", doc2))
	require.Equal(t, []string{"y1", "y2", "x1", "x2"}, evalIDs(t, ctx2, eval, "/a/b | $other/b", doc1))
}

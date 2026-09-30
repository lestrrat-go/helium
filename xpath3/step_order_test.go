package xpath3_test

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
	"github.com/lestrrat-go/helium/enum"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// stepOrderGoldenPath holds the recorded result of every case in
// TestStepResultOrder. Set XPATH3_UPDATE_STEP_ORDER=1 to rewrite it.
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
// $nodes and $other variables so cross-document sequences are covered.
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
// a sequence drawn from the content of several entities.
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
	for n := range helium.Descendants(doc) {
		if n.Type() == helium.EntityRefNode {
			return n
		}
	}
	return nil
}

// pickEntity returns the Entity node the first entity reference of doc
// refers to, or nil.
func pickEntity(doc *helium.Document) helium.Node {
	ref := pickEntityRef(doc)
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

// exprDescendantOrSelfNode is the step that `//` abbreviates.
const exprDescendantOrSelfNode = "descendant-or-self::node()"

// stepOrderPathExprs are path expressions whose result XPath 3.1 defines to be
// in document order without duplicates, so every result is also checked
// against a fresh document-order index.
var stepOrderPathExprs = []string{
	// Axis-step shapes shared with xpath1/step_order_test.go.
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
	"$nodes/b", "$nodes//b", "$nodes/..", "/a/b | $other/b", "//b | $other//b",
	"//b | ($other/b | $other/c)", "$other/b | /a/b", "($nodes)[1]//c",
	exprDescendantOrSelfNode, ".//node()", ".//comment()", ".//processing-instruction()", ".//text()",
	"descendant::node()/..", "child::node()", "self::node()", "descendant::x",
	"$ents/node()", "$ents/*/node()", "$ents/..", "$ents/self::*", "$ents/@*", "$ents/namespace::*",
	"$ents/following-sibling::node()", "$ents/following::node()", "$ents/preceding::node()",
	"$ents/descendant::node()", "$ents/ancestor::node()", "$ents | //b", "$ents | $ents/node()",
	"following::*[count(ancestor::node()) = 3]", "following::*[count(ancestor::node()) = 3]/node()",
	"following::*[count(ancestor::node()) = 3]/@*", "following::*[count(ancestor::node()) = 3]/node()/..",
	// Positional predicates on forward and reverse axes.
	"(//b)[last()]/preceding::*", "//b/preceding-sibling::*[1]", "//b/ancestor::*[1]",
	"ancestor::*[1]", "ancestor::*[last()]", "//a/b[1]", "//b/@*[1]", "//*[@id = 'b1']/..",
	"descendant::*/namespace::*", "namespace::*", "//b/following::*[1]", "//b/self::node()[@id]",
	"//a/node()[last()]", "//a/b/following-sibling::node()[1]/..", "//b/descendant-or-self::node()[1]",
	// Kind tests.
	"//a/element()", "//a/attribute()", "//a/child::element(b)",
	// Path steps whose step expression is not an axis step.
	"/(a|b)", "//a/(b|c)", "//b/(.., c)", "//b/(@id, ..)", "//b/reverse(ancestor::*)",
	"//b/(if (@id) then . else ..)", "$nodes/(b | c)", "//a/$other", "//b/root()", "//b/root(.)/a",
	"//b/..[1]", "//a/(b)", "//*/(.)", "//b/id('b1')", "//b/(ancestor::*)[1]", "//b/(ancestor::*[1])",
	"//a/b/(c|x)/..", "//b/(following::*)[1]", "//b/(following::* | preceding::*)", "$ents/(..)",
	"$ents/(node())", "//b/(1, 2)[. = 0]",
	// Set operators.
	"//a intersect //b/..", "//a except //b/..",
}

// stepOrderOtherExprs are expressions whose result is not a document-ordered
// node sequence: the simple map operator, atomic last steps, explicit
// reordering, FLWOR and sequence construction, and mixed node/atomic step
// results (XPTY0018). They are pinned by the golden file only.
var stepOrderOtherExprs = []string{
	"//a ! b", "//a ! ..", "//b ! string(@id)", "//b ! (.., .)", "//b/string(@id)", "//b/data(@id)",
	"/a/(1, .)", "//b/(if (@id) then . else 'x')", "//b/(., 'x')", "reverse(//b)/self::b", "reverse(//b)/c",
	"reverse(//b)/..", "for $x in //b return $x/..", "(//b, //a)", "(//b, //a)/.", "//b/count(ancestor::*)",
	"//b/name()", "sort(//b, (), function($n) { string($n/@id) })/..", "//b/position()",
	"(/a/b)/local-name()", "$nodes | //b", "//b | $nodes", "$nodes | $other",
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
	labels map[helium.Node]string
	eval   xpath3.Evaluator
}

func nodeSequence(nodes []helium.Node) xpath3.Sequence {
	seq := make(xpath3.ItemSlice, len(nodes))
	for i, n := range nodes {
		seq[i] = xpath3.NodeItem{Node: n}
	}
	return seq
}

func newStepOrderFixture(t testing.TB, d stepOrderDoc) stepOrderFixture {
	t.Helper()
	doc := parseStepOrderDoc(t, d.src, d.subst)
	if d.build != nil {
		d.build(t, doc)
	}
	other := parseStepOrderDoc(t, stepOrderOtherDoc, false)
	labels := map[helium.Node]string{doc: "d1:", other: "d2:"}
	vars := map[string]xpath3.Sequence{
		"nodes": nodeSequence([]helium.Node{other.DocumentElement(), doc.DocumentElement()}),
		"other": nodeSequence([]helium.Node{other.DocumentElement()}),
		"ents":  nodeSequence(entityElems(t, doc)),
	}
	return stepOrderFixture{
		doc:    doc,
		labels: labels,
		eval:   xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(vars),
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
	r, err := f.eval.Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(expr), f.doc)
	require.NoError(t, err)
	nodes, err := r.Nodes()
	require.NoError(t, err)
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

// describeResult evaluates expr and renders its result sequence. When
// ordered is set and every item is a node, it also checks that the nodes are
// already sorted and duplicate-free according to a fresh document-order
// index (the order oracle).
func (f stepOrderFixture) describeResult(t testing.TB, ctxNode helium.Node, expr string, ordered bool) string {
	t.Helper()
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err, expr)
	r, err := f.eval.Evaluate(t.Context(), compiled, ctxNode)
	if err != nil {
		return "ERR " + err.Error()
	}
	seq := r.Sequence()
	nodes, allNodes := xpath3.NodesFrom(seq)
	if !allNodes {
		return f.describeItems(seq)
	}
	got := f.describeNodes(nodes)
	if !ordered {
		return got
	}
	sorted, err := ixpath.DeduplicateNodes(slices.Clone(nodes), &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	if !slices.Equal(nodes, sorted) {
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

// describeItems renders a sequence that holds at least one non-node item.
func (f stepOrderFixture) describeItems(seq xpath3.Sequence) string {
	var parts []string
	for item := range seq.Items() {
		switch v := item.(type) {
		case xpath3.NodeItem:
			parts = append(parts, describeNode(v.Node, f.labels))
		case xpath3.AtomicValue:
			parts = append(parts, v.String())
		default:
			parts = append(parts, fmt.Sprintf("%T", item))
		}
	}
	return strings.Join(parts, " ")
}

// TestStepResultOrder pins the result sequence and its order for a matrix of
// documents, context nodes and expressions against a recorded golden file,
// so any change to how a path step orders or deduplicates its result shows
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
			for _, expr := range stepOrderPathExprs {
				fmt.Fprintf(&got, "%s|%s|%s => %s\n", d.name, c.name, expr, f.describeResult(t, ctxNode, expr, true))
			}
			for _, expr := range stepOrderOtherExprs {
				fmt.Fprintf(&got, "%s|%s|%s => %s\n", d.name, c.name, expr, f.describeResult(t, ctxNode, expr, false))
			}
		}
	}

	if os.Getenv("XPATH3_UPDATE_STEP_ORDER") == "1" {
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

// TestStepResultOrderRandom runs the path expressions over generated
// documents and checks every node sequence against a fresh document-order
// index.
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
			for _, expr := range stepOrderPathExprs {
				f.describeResult(t, ctxNode, expr, true)
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

// stepOrderIDs evaluates expr from ctxNode and returns the id attribute (or
// the element name when there is none) of every selected node, in result
// order.
func stepOrderIDs(t *testing.T, eval xpath3.Evaluator, expr string, ctxNode helium.Node) []string {
	t.Helper()
	r, err := eval.Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(expr), ctxNode)
	require.NoError(t, err)
	nodes, err := r.Nodes()
	require.NoError(t, err)
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
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
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	t.Run("fresh cache per evaluation", func(t *testing.T) {
		doc := parseStepOrderDoc(t, src, false)
		require.Equal(t, []string{"i1", "i2", "i3"}, stepOrderIDs(t, eval, "//item", doc))
		first := doc.DocumentElement().FirstChild().(*helium.Element)
		moveToEnd(t, first)
		require.Equal(t, []string{"i2", "i3", "i1"}, stepOrderIDs(t, eval, "//item", doc))
		require.Equal(t, []string{"i2", "i3", "o", "i1"}, stepOrderIDs(t, eval, "/root/item | /root/other", doc))
		helium.UnlinkNode(first)
		require.Equal(t, []string{"i2", "i3"}, stepOrderIDs(t, eval, "/root/item", doc))
	})

	t.Run("caller-supplied cache reset after the mutation", func(t *testing.T) {
		doc := parseStepOrderDoc(t, src, false)
		cache := xpath3.NewDocOrderCache()
		shared := eval.DocOrderCache(cache)
		require.Equal(t, []string{"i1", "i2", "i3"}, stepOrderIDs(t, shared, "/root/item", doc))
		require.Equal(t, []string{"i1", "i2", "i3", "o"}, stepOrderIDs(t, shared, "/root/item | /root/other", doc))

		moveToEnd(t, doc.DocumentElement().FirstChild().(*helium.Element))
		cache.Reset()
		require.Equal(t, []string{"i2", "i3", "i1"}, stepOrderIDs(t, shared, "/root/item", doc))
		require.Equal(t, []string{"i2", "i3", "o", "i1"}, stepOrderIDs(t, shared, "/root/item | /root/other", doc))
	})

	t.Run("caller-supplied cache reset before any union", func(t *testing.T) {
		doc := parseStepOrderDoc(t, src, false)
		cache := xpath3.NewDocOrderCache()
		shared := eval.DocOrderCache(cache)
		require.Equal(t, []string{"i1", "i2", "i3"}, stepOrderIDs(t, shared, "/root/item", doc))

		moveToEnd(t, doc.DocumentElement().FirstChild().(*helium.Element))
		cache.Reset()
		require.Equal(t, []string{"i2", "i3", "o", "i1"}, stepOrderIDs(t, shared, "/root/item | /root/other", doc))
	})
}

// TestStepResultOrderSharedCache checks a caller-supplied cache across
// evaluations over two documents: later evaluations order the documents by
// the order in which the cache first met them.
func TestStepResultOrderSharedCache(t *testing.T) {
	doc1 := parseStepOrderDoc(t, `<a><b id="x1"><c id="xc"/></b><b id="x2"/></a>`, false)
	doc2 := parseStepOrderDoc(t, `<a><b id="y1"/><b id="y2"><c id="yc"/></b></a>`, false)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(map[string]xpath3.Sequence{
		"other": nodeSequence([]helium.Node{doc2.DocumentElement()}),
	})
	shared := eval.DocOrderCache(xpath3.NewDocOrderCache())

	require.Equal(t, []string{"x1", "x2"}, stepOrderIDs(t, shared, "/a/b", doc1))

	// doc1 was met first, so its nodes sort before doc2's.
	require.Equal(t, []string{"x1", "x2", "y1", "y2"}, stepOrderIDs(t, shared, "$other/b | /a/b", doc1))
	require.Equal(t, []string{"x1", "xc", "x2", "y1", "y2", "yc"}, stepOrderIDs(t, shared, "$other//b | $other//c | /a/b/c | /a/b", doc1))
	require.Equal(t, []string{"x1", "xc", "x2"}, stepOrderIDs(t, shared, "/a/b/c | /a/b", doc1))

	// A second cache that meets doc2 first orders doc2 first.
	shared2 := eval.DocOrderCache(xpath3.NewDocOrderCache())
	require.Equal(t, []string{"y1", "y2"}, stepOrderIDs(t, shared2, "/a/b", doc2))
	require.Equal(t, []string{"y1", "y2", "x1", "x2"}, stepOrderIDs(t, shared2, "/a/b | $other/b", doc1))
}

// TestStepResultOrderNodeLimit checks that the node-sequence limit still
// applies to steps whose result needs no sort: child steps from one input,
// attribute steps from same-depth inputs, and a reverse axis from one input,
// each with and without a predicate.
func TestStepResultOrderNodeLimit(t *testing.T) {
	doc := parseStepOrderDoc(t, `<root><item id="i1" k="1"><v/></item><item id="i2" k="2"><v/></item><item id="i3"><v/></item></root>`, false)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).MaxNodesForTesting(2)
	v := doc.DocumentElement().FirstChild().FirstChild()
	cases := []struct {
		expr string
		node helium.Node
	}{
		{expr: "/root/item", node: doc},
		{expr: "/root/item[@id]", node: doc},
		{expr: "/root/item[position() <= 2]/@*", node: doc},
		{expr: "/root/item[position() <= 2]/@*[true()]", node: doc},
		{expr: "ancestor-or-self::node()", node: v},
		{expr: "ancestor-or-self::node()[. instance of element()]", node: v},
	}
	for _, c := range cases {
		_, err := eval.Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(c.expr), c.node)
		require.ErrorIs(t, err, xpath3.ErrNodeSetLimit, c.expr)
	}
	r, err := eval.Evaluate(t.Context(), xpath3.NewCompiler().MustCompile("/root/item[position() <= 2]/@id"), doc)
	require.NoError(t, err)
	nodes, err := r.Nodes()
	require.NoError(t, err)
	require.Len(t, nodes, 2)
}

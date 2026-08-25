package xslt3

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// dispatchIndexPatterns is a representative set of XSLT match patterns
// covering every AST form the dispatch index's signature extractor
// (patternSignature / stepSignature in dispatch_index.go) has to classify —
// both the forms it can safely narrow (LocationPath, PathStepExpr, RootExpr,
// UnionExpr) and the forms it must conservatively treat as unindexed
// (ContextItemExpr, FilterExpr, VariableExpr, IntersectExceptExpr, PathExpr).
// The stylesheet under test (dispatchIndexTestStylesheet) registers one
// xsl:template per entry, in mode "test".
var dispatchIndexPatterns = []string{
	// Plain element / attribute name tests (byName candidates).
	"foo",
	"bar",
	"@id",
	"@class",
	// Wildcards (byKind candidates: name is not concrete).
	"*",
	"@*",
	"ns:*",
	"*:foo",
	"@ns:*",
	// Prefixed and EQName element names.
	"ns:foo",
	"Q{http://example.com/ns}foo",
	// Kind tests.
	"text()",
	"comment()",
	"processing-instruction()",
	"processing-instruction('target')",
	"node()",
	"self::node()",
	"namespace::node()",
	// element()/attribute() kind tests (kind bucket, not name bucket).
	"element(foo)",
	"element()",
	"attribute(id)",
	// Document tests.
	"/",
	"/foo",
	"document-node()",
	// Multi-step LocationPath (terminal step governs the signature).
	"foo/bar",
	"foo//bar",
	"*/bar",
	"foo/@id",
	// Predicates narrow but never widen — signature is the same as "foo".
	"foo[@id]",
	"foo[position() = 1]",
	"bar[ns:foo]",
	// Union patterns: same-template split candidates (no explicit priority)...
	"foo | quux",
	// ...and explicit-priority unions that stay ONE template with multiple
	// Alternatives (compile_templates.go only splits when priority is absent).
	`bar | quux2`,
	// Schema kind tests (kind bucket; always false without a schema, so the
	// soundness property holds trivially but the signature still has to
	// classify the AST without panicking).
	"schema-element(foo)",
	"schema-attribute(id)",
	// Unindexed forms.
	".",
	".[position() = 1]",
	"$g",
	"foo intersect bar",
	"foo except bar",
	"key('k', 'v')",
	"key('k', 'v')/foo",
	"id('x')",
	"id('x')/foo",
}

// dispatchIndexTestStylesheet builds a stylesheet source registering one
// xsl:template per pattern in dispatchIndexPatterns, in mode "test", plus the
// global variable/key declarations some of those patterns reference.
func dispatchIndexTestStylesheet(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>` + "\n")
	b.WriteString(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:ns="http://example.com/ns">` + "\n")
	b.WriteString(`<xsl:variable name="g" select="1"/>` + "\n")
	b.WriteString(`<xsl:key name="k" match="foo" use="@id"/>` + "\n")
	for i, pat := range dispatchIndexPatterns {
		esc := strings.ReplaceAll(pat, "&", "&amp;")
		esc = strings.ReplaceAll(esc, "<", "&lt;")
		esc = strings.ReplaceAll(esc, `"`, "&quot;")
		priority := ""
		if pat == `bar | quux2` {
			priority = ` priority="5"`
		}
		fmt.Fprintf(&b, `<xsl:template match="%s" mode="test"%s><hit n="%d"/></xsl:template>`+"\n", esc, priority, i)
	}
	b.WriteString(`</xsl:stylesheet>`)
	return b.String()
}

// dispatchIndexTestSource is a small document exercising every node kind the
// signature extractor distinguishes: elements (named and namespaced), an
// attribute, text, a comment, and a processing instruction.
const dispatchIndexTestSource = `<?xml version="1.0"?>
<root xmlns:ns="http://example.com/ns" id="r1">
  <foo id="f1">text content<bar/><ns:foo/></foo>
  <bar class="b"><!--a comment--><?target data?></bar>
  <ns:foo id="nf1"/>
  <quux/>
  <quux2/>
  <nested/>
</root>`

// dispatchIndexTestExecContext builds a minimal but functional execContext
// sufficient for pattern.matchPattern: it needs a stylesheet, a source
// document (for base-URI / doc() lookups an evaluated predicate might touch),
// and the caches xpathEvaluator/evalXPath read (docOrderCache, key tables,
// global vars). It does not need result-document / output plumbing since the
// property test never executes a template body.
func dispatchIndexTestExecContext(ss *Stylesheet, source *helium.Document) *execContext {
	ec := &execContext{
		stylesheet:          ss,
		sourceDoc:           source,
		globalVars:          make(map[string]xpath3.Sequence),
		keyTables:           make(map[string]*keyTable),
		docCache:            make(map[string]*helium.Document),
		functionResultCache: make(map[string]xpath3.Sequence),
		currentTime:         time.Now().UTC(),
		defaultValidation:   ss.defaultValidation,
		defaultCollation:    ss.defaultCollation,
		docOrderCache:       xpath3.NewDocOrderCache(),
	}
	ec.setCurrentTemplate(nil)
	return ec
}

// allNodes collects every node in doc order under root, plus root itself:
// elements, their attributes, and every child (text, comment, PI, nested
// elements).
func allNodes(root helium.Node) []helium.Node {
	var out []helium.Node
	var walk func(n helium.Node)
	walk = func(n helium.Node) {
		out = append(out, n)
		if elem, ok := n.(*helium.Element); ok {
			for _, attr := range elem.Attributes() {
				out = append(out, attr)
			}
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(root)
	return out
}

// TestDispatchIndexSignatureSoundness is the property test required before
// the dispatch index exists (design step 2): for every (template, node) pair
// drawn from dispatchIndexPatterns and every node of dispatchIndexTestSource,
// signatureAdmits(tmpl, node) must be true whenever matchPattern(node) is
// true. Equivalently: a node the signature says CANNOT match must genuinely
// not match. This is the soundness property the whole index rests on — the
// index only ever narrows a scan to positions signatureAdmits accepts, so a
// false "cannot match" would silently drop a real match.
func TestDispatchIndexSignatureSoundness(t *testing.T) {
	src := dispatchIndexTestStylesheet(t)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	ss, err := compile(t.Context(), doc, &compileConfig{})
	require.NoError(t, err)

	sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(dispatchIndexTestSource))
	require.NoError(t, err)
	docRoot := helium.Node(sourceDoc)

	// len(templates) is not exactly len(dispatchIndexPatterns): an
	// unprioritized union pattern ("foo | quux") is split into one template
	// per alternative at compile time (compile_templates.go), so it
	// contributes two entries here.
	templates := ss.modeTemplates["test"]
	require.GreaterOrEqual(t, len(templates), len(dispatchIndexPatterns))

	nodes := allNodes(docRoot)
	require.NotEmpty(t, nodes)

	ec := dispatchIndexTestExecContext(ss, sourceDoc)

	checked := 0
	for _, tmpl := range templates {
		for _, node := range nodes {
			admits := signatureAdmits(tmpl, node)
			matches := tmpl.Match.matchPattern(t.Context(), ec, node)
			if matches {
				require.Truef(t, admits,
					"pattern %q matched a node the signature ruled out (kind=%v)",
					tmpl.Match.source, node.Type())
			}
			checked++
		}
	}
	require.Positive(t, checked)
}

// TestIndexCursor exercises the merge cursor (design step 5) directly against
// hand-built [3][]int32 lists, independent of templateIndex construction: it
// must reproduce the exact ascending subsequence, dedup a position repeated
// across lists or within one list, and allocate nothing.
func TestIndexCursor(t *testing.T) {
	drain := func(c indexCursor) []int32 {
		var got []int32
		for {
			pos, ok := c.next()
			if !ok {
				return got
			}
			got = append(got, pos)
		}
	}

	t.Run("merges three ascending lists in order", func(t *testing.T) {
		c := indexCursor{lists: [3][]int32{
			{1, 4, 9},
			{2, 4, 7},
			{0, 4, 5, 9},
		}}
		require.Equal(t, []int32{0, 1, 2, 4, 5, 7, 9}, drain(c))
	})

	t.Run("dedups a value repeated within one list", func(t *testing.T) {
		c := indexCursor{lists: [3][]int32{
			{3, 3, 3, 5},
			nil,
			nil,
		}}
		require.Equal(t, []int32{3, 5}, drain(c))
	})

	t.Run("a single non-empty list passes through unchanged", func(t *testing.T) {
		c := indexCursor{lists: [3][]int32{nil, {2, 6, 8}, nil}}
		require.Equal(t, []int32{2, 6, 8}, drain(c))
	})

	t.Run("every list empty produces no candidates", func(t *testing.T) {
		var c indexCursor
		_, ok := c.next()
		require.False(t, ok)
	})

	t.Run("allocates nothing", func(t *testing.T) {
		lists := [3][]int32{{1, 2, 3}, {2, 4}, {0, 5}}
		allocs := testing.AllocsPerRun(1000, func() {
			c := indexCursor{lists: lists}
			for {
				if _, ok := c.next(); !ok {
					break
				}
			}
		})
		require.Zero(t, allocs)
	})
}

// TestCandidates exercises the candidates() entry point (design step 5)
// against a hand-built templateIndex and real nodes from a small parsed
// document, verifying the byName / byKind / unindexed buckets are selected
// correctly for each node kind.
func TestCandidates(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<root xmlns:ns="http://example.com/ns" id="r1"><foo/><ns:foo/>text<!--c--></root>`))
	require.NoError(t, err)
	root := doc.DocumentElement()

	var fooElem, nsFooElem *helium.Element
	var idAttr *helium.Attribute
	var textNode, commentNode helium.Node
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *helium.Element:
			if v.URI() == "" {
				fooElem = v
			} else {
				nsFooElem = v
			}
		case *helium.Text:
			textNode = v
		case *helium.Comment:
			commentNode = v
		}
	}
	idAttr = root.Attributes()[0]
	require.NotNil(t, fooElem)
	require.NotNil(t, nsFooElem)
	require.NotNil(t, idAttr)
	require.NotNil(t, textNode)
	require.NotNil(t, commentNode)

	idx := &templateIndex{
		byName: map[nameKey][]int32{
			{kind: kindElement, local: "foo"}:                               {1, 5},
			{kind: kindElement, uri: "http://example.com/ns", local: "foo"}: {2},
			{kind: kindAttribute, local: "id"}:                              {3},
		},
		unindexed: []int32{0, 9},
	}
	idx.byKind[kindText] = []int32{4}

	require.Equal(t, []int32{0, 1, 5, 9}, drainCandidates(idx, fooElem))
	require.Equal(t, []int32{0, 2, 9}, drainCandidates(idx, nsFooElem))
	require.Equal(t, []int32{0, 3, 9}, drainCandidates(idx, idAttr))
	require.Equal(t, []int32{0, 4, 9}, drainCandidates(idx, textNode))
	// A comment falls in neither byName nor any populated byKind bucket for
	// kindComment — only the unindexed bucket admits it.
	require.Equal(t, []int32{0, 9}, drainCandidates(idx, commentNode))
}

func drainCandidates(idx *templateIndex, node helium.Node) []int32 {
	c := candidates(idx, node)
	var got []int32
	for {
		pos, ok := c.next()
		if !ok {
			return got
		}
		got = append(got, pos)
	}
}
